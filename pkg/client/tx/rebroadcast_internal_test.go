package tx

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// newTestTxClientForRebroadcast builds a bare txClient with only the fields that
// collectDueRebroadcasts touches, so the re-broadcast gate logic can be tested in
// isolation (no mocks, no goroutines).
func newTestTxClientForRebroadcast(pool map[txHash]*pendingRebroadcast) *txClient {
	return &txClient{rebroadcastPool: pool}
}

func TestCollectDueRebroadcasts_SpacingAndSafetyGates(t *testing.T) {
	// A 10-block window: broadcast at height 10, timeout (window close) at height 20.
	// Re-broadcasts are spread evenly: with maxTxRebroadcasts=2 the first is due at
	// 10 + 10*1/3 = 13. The safety margin stops resends at height >= 19. Each subtest
	// uses a fresh tx (rebroadcasts=0), so it gates whether the FIRST resend fires.
	const (
		submitHeight  = int64(10)
		timeoutHeight = int64(20)
	)

	tests := []struct {
		name      string
		height    int64
		expectDue bool
	}{
		{name: "before first resend point: no resend", height: 12, expectDue: false},
		{name: "at first resend point: resend", height: 13, expectDue: true},
		{name: "past first resend point: resend", height: 16, expectDue: true},
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

func TestCollectDueRebroadcasts_EvenlySpacedAcrossWindow(t *testing.T) {
	// Window 10-20 (size 10), maxTxRebroadcasts=2: resend points at 13 (1/3) and 16 (2/3).
	pool := map[txHash]*pendingRebroadcast{
		"hash-a": {txBz: []byte("tx-a"), submitHeight: 10, timeoutHeight: 20},
	}
	txnClient := newTestTxClientForRebroadcast(pool)

	// Before the first point: nothing due.
	require.Empty(t, txnClient.collectDueRebroadcasts(12))

	// First point (1/3 of the window): one resend, counter -> 1.
	first := txnClient.collectDueRebroadcasts(13)
	require.Len(t, first, 1)
	require.Equal(t, 1, pool["hash-a"].rebroadcasts)

	// Between the two points: the second is not due yet.
	require.Empty(t, txnClient.collectDueRebroadcasts(14))

	// Second point (2/3 of the window): one resend, counter -> 2.
	second := txnClient.collectDueRebroadcasts(16)
	require.Len(t, second, 1)
	require.Equal(t, 2, pool["hash-a"].rebroadcasts)
}

func TestCollectDueRebroadcasts_BoundedByMaxRebroadcasts(t *testing.T) {
	pool := map[txHash]*pendingRebroadcast{
		"hash-a": {txBz: []byte("tx-a"), submitHeight: 10, timeoutHeight: 20},
	}
	txnClient := newTestTxClientForRebroadcast(pool)

	// Each due collection past a resend point returns the tx once and increments the
	// counter; it takes maxTxRebroadcasts collections to reach the cap.
	for i := 1; i <= maxTxRebroadcasts; i++ {
		// Collect at the timeout-1 safe edge so every remaining resend point is due.
		due := txnClient.collectDueRebroadcasts(18)
		require.Len(t, due, 1)
		require.Equal(t, i, pool["hash-a"].rebroadcasts)
	}
	require.Equal(t, maxTxRebroadcasts, pool["hash-a"].rebroadcasts)

	// Once the cap is reached, a later, still-valid block must NOT resend again.
	require.Empty(t, txnClient.collectDueRebroadcasts(18))
}

func TestCollectDueRebroadcasts_OnlyDueEntriesReturned(t *testing.T) {
	txnClient := newTestTxClientForRebroadcast(map[txHash]*pendingRebroadcast{
		// First resend due at height 13 (window 10-20, 1/3 point).
		"due": {txBz: []byte("due"), submitHeight: 10, timeoutHeight: 20},
		// First resend not due until 12 + (40-12)/3 = 21.
		"not-yet": {txBz: []byte("not-yet"), submitHeight: 12, timeoutHeight: 40},
	})

	due := txnClient.collectDueRebroadcasts(13)
	require.Len(t, due, 1)
	require.Equal(t, "due", due[0].txHash)
}
