package tx

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// newTestTxClientForRebroadcast builds a bare txClient with only the fields that
// collectDueRebroadcasts touches, so the re-broadcast gate logic can be tested in
// isolation (no mocks, no goroutines).
func newTestTxClientForRebroadcast(pool map[txHash]*pendingRebroadcast) *txClient {
	return &txClient{rebroadcastPool: pool}
}

// expectedDueHeight mirrors the scheduling math in collectDueRebroadcasts for the
// k-th (1-based) re-broadcast of a single tx, including the per-tx jitter and the
// safety clamp. Tests use it so they assert the gate semantics (before/at/after,
// cap, safety) without hard-coding a specific hash's jitter offset.
func expectedDueHeight(hash txHash, p *pendingRebroadcast, k int64) int64 {
	window := p.timeoutHeight - p.submitHeight
	slot := window / int64(maxTxRebroadcasts+1)
	due := p.submitHeight + window*k/int64(maxTxRebroadcasts+1) + rebroadcastJitter(hash, slot)
	if maxDue := p.timeoutHeight - txRebroadcastSafetyBlocks - 1; due > maxDue {
		due = maxDue
	}
	return due
}

func TestCollectDueRebroadcasts_SpacingAndSafetyGates(t *testing.T) {
	// A 10-block window: broadcast at height 10, timeout (window close) at height 20.
	// The FIRST resend is due at its (jittered) 1/3 point; the safety margin stops
	// resends at height >= 19. Each subtest uses a fresh tx (rebroadcasts=0), so it
	// gates whether the FIRST resend fires.
	const (
		submitHeight  = int64(10)
		timeoutHeight = int64(20)
	)
	pending := &pendingRebroadcast{txBz: []byte("tx-a"), submitHeight: submitHeight, timeoutHeight: timeoutHeight}
	firstDue := expectedDueHeight("hash-a", pending, 1)

	tests := []struct {
		name      string
		height    int64
		expectDue bool
	}{
		{name: "one block before first resend point: no resend", height: firstDue - 1, expectDue: false},
		{name: "at first resend point: resend", height: firstDue, expectDue: true},
		{name: "past first resend point: resend", height: firstDue + 1, expectDue: true},
		{name: "within safety margin of timeout: no resend", height: 19, expectDue: false},
		{name: "at timeout height: no resend", height: 20, expectDue: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			txnClient := newTestTxClientForRebroadcast(map[txHash]*pendingRebroadcast{
				"hash-a": {txBz: []byte("tx-a"), submitHeight: submitHeight, timeoutHeight: timeoutHeight},
			})

			due := txnClient.collectDueRebroadcasts(test.height)

			if test.expectDue {
				require.Len(t, due, 1)
				require.Equal(t, "hash-a", due[0].txHash)
				require.Equal(t, []byte("tx-a"), due[0].txBz)
			} else {
				require.Empty(t, due)
			}
		})
	}
}

func TestCollectDueRebroadcasts_SpacedAcrossWindow(t *testing.T) {
	// Window 10-20 (size 10), maxTxRebroadcasts=2: two resend points, each jittered
	// within its slot for this hash.
	pool := map[txHash]*pendingRebroadcast{
		"hash-a": {txBz: []byte("tx-a"), submitHeight: 10, timeoutHeight: 20},
	}
	firstDue := expectedDueHeight("hash-a", pool["hash-a"], 1)
	secondDue := expectedDueHeight("hash-a", pool["hash-a"], 2)
	require.Less(t, firstDue, secondDue, "the two resend points must remain ordered after jitter")

	txnClient := newTestTxClientForRebroadcast(pool)

	// Before the first point: nothing due.
	require.Empty(t, txnClient.collectDueRebroadcasts(firstDue-1))

	// First point: one resend, counter -> 1.
	first := txnClient.collectDueRebroadcasts(firstDue)
	require.Len(t, first, 1)
	require.Equal(t, 1, pool["hash-a"].rebroadcasts)

	// Between the two points: the second is not due yet.
	require.Empty(t, txnClient.collectDueRebroadcasts(secondDue-1))

	// Second point: one resend, counter -> 2.
	second := txnClient.collectDueRebroadcasts(secondDue)
	require.Len(t, second, 1)
	require.Equal(t, 2, pool["hash-a"].rebroadcasts)
}

func TestCollectDueRebroadcasts_BoundedByMaxRebroadcasts(t *testing.T) {
	pool := map[txHash]*pendingRebroadcast{
		"hash-a": {txBz: []byte("tx-a"), submitHeight: 10, timeoutHeight: 20},
	}
	txnClient := newTestTxClientForRebroadcast(pool)

	// Collect at the timeout-1 safe edge (18) so every remaining resend point is due.
	// Each collection returns the tx once and increments the counter; it takes
	// maxTxRebroadcasts collections to reach the cap.
	for i := 1; i <= maxTxRebroadcasts; i++ {
		due := txnClient.collectDueRebroadcasts(18)
		require.Len(t, due, 1)
		require.Equal(t, i, pool["hash-a"].rebroadcasts)
	}
	require.Equal(t, maxTxRebroadcasts, pool["hash-a"].rebroadcasts)

	// Once the cap is reached, a later, still-valid block must NOT resend again.
	require.Empty(t, txnClient.collectDueRebroadcasts(18))
}

// TestCollectDueRebroadcasts_JitterFansOutBatch is the point of the jitter: a whole
// claim/proof batch shares the same submitHeight and timeoutHeight, and without a
// per-tx offset every tx would re-broadcast on the exact same block (a synchronized
// CheckTx burst on the node). This verifies the batch is spread across more than one
// block, and that no tx is scheduled at or past the safety boundary.
func TestCollectDueRebroadcasts_JitterFansOutBatch(t *testing.T) {
	const (
		submitHeight  = int64(100)
		timeoutHeight = int64(130) // window 30 -> slot 10, room to observe spread
	)
	safetyEdge := timeoutHeight - txRebroadcastSafetyBlocks // resends must be strictly below this

	// A batch of txs identical except for their hash key.
	pool := make(map[txHash]*pendingRebroadcast)
	const batchSize = 50
	for i := 0; i < batchSize; i++ {
		pool[fmt.Sprintf("claim-%d", i)] = &pendingRebroadcast{
			txBz:          []byte(fmt.Sprintf("tx-%d", i)),
			submitHeight:  submitHeight,
			timeoutHeight: timeoutHeight,
		}
	}

	// Compute each tx's first-resend height and bucket by block.
	firstDueByHeight := make(map[int64]int)
	for hash, p := range pool {
		due := expectedDueHeight(hash, p, 1)
		require.Less(t, due, safetyEdge, "jitter must never push a resend to/past the safety boundary")
		require.GreaterOrEqual(t, due, submitHeight, "resend must never precede submit")
		firstDueByHeight[due]++
	}

	// The whole batch must NOT collapse onto a single block.
	require.Greater(t, len(firstDueByHeight), 1,
		"jitter should fan the batch across multiple blocks, got all on %d block(s)", len(firstDueByHeight))

	// Draining the whole window must eventually resend every tx exactly
	// maxTxRebroadcasts times (jitter changes when, never whether).
	txnClient := newTestTxClientForRebroadcast(pool)
	resendCount := make(map[txHash]int)
	for h := submitHeight; h < safetyEdge; h++ {
		for _, item := range txnClient.collectDueRebroadcasts(h) {
			resendCount[item.txHash]++
		}
	}
	require.Len(t, resendCount, batchSize)
	for hash, n := range resendCount {
		require.Equal(t, maxTxRebroadcasts, n, "tx %q resent %d times, want %d", hash, n, maxTxRebroadcasts)
	}
}

func TestCollectDueRebroadcasts_OnlyDueEntriesReturned(t *testing.T) {
	pool := map[txHash]*pendingRebroadcast{
		"due":     {txBz: []byte("due"), submitHeight: 10, timeoutHeight: 20},
		"not-yet": {txBz: []byte("not-yet"), submitHeight: 12, timeoutHeight: 40},
	}
	txnClient := newTestTxClientForRebroadcast(pool)

	dueHeight := expectedDueHeight("due", pool["due"], 1)
	notYetHeight := expectedDueHeight("not-yet", pool["not-yet"], 1)
	require.Less(t, dueHeight, notYetHeight, "test setup: 'due' must fire before 'not-yet'")

	due := txnClient.collectDueRebroadcasts(dueHeight)
	require.Len(t, due, 1)
	require.Equal(t, "due", due[0].txHash)
}
