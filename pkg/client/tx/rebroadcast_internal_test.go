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

func TestCollectDueRebroadcasts_MidpointAndSafetyGates(t *testing.T) {
	// A 10-block window: broadcast at height 10, timeout (window close) at height 20.
	// midpoint = 10 + (20-10)/2 = 15; safety margin stops resends at height >= 19.
	const (
		submitHeight  = int64(10)
		timeoutHeight = int64(20)
	)

	tests := []struct {
		name      string
		height    int64
		expectDue bool
	}{
		{name: "before midpoint: no resend", height: 14, expectDue: false},
		{name: "at midpoint: resend", height: 15, expectDue: true},
		{name: "mid window: resend", height: 17, expectDue: true},
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

func TestCollectDueRebroadcasts_BoundedByMaxRebroadcasts(t *testing.T) {
	pool := map[txHash]*pendingRebroadcast{
		"hash-a": {txBz: []byte("tx-a"), submitHeight: 10, timeoutHeight: 20},
	}
	txnClient := newTestTxClientForRebroadcast(pool)

	// First due collection past the midpoint returns the tx and increments the count.
	first := txnClient.collectDueRebroadcasts(15)
	require.Len(t, first, 1)
	require.Equal(t, maxTxRebroadcasts, pool["hash-a"].rebroadcasts)

	// A later, still-valid block must NOT resend again once the cap is reached.
	second := txnClient.collectDueRebroadcasts(17)
	require.Empty(t, second)
}

func TestCollectDueRebroadcasts_OnlyDueEntriesReturned(t *testing.T) {
	txnClient := newTestTxClientForRebroadcast(map[txHash]*pendingRebroadcast{
		// Due at height 15 (window 10-20, midpoint 15).
		"due": {txBz: []byte("due"), submitHeight: 10, timeoutHeight: 20},
		// Not yet at its midpoint (12 + (40-12)/2 = 26).
		"not-yet": {txBz: []byte("not-yet"), submitHeight: 12, timeoutHeight: 40},
	})

	due := txnClient.collectDueRebroadcasts(15)
	require.Len(t, due, 1)
	require.Equal(t, "due", due[0].txHash)
}
