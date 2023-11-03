package block_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	comettypes "github.com/cometbft/cometbft/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/internal/testclient"
	"github.com/pokt-network/poktroll/internal/testclient/testeventsquery"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/block"
	eventsquery "github.com/pokt-network/poktroll/pkg/client/events_query"
)

const testTimeoutDuration = 100 * time.Millisecond

func TestBlockClient(t *testing.T) {
	var (
		expectedHeight     = int64(1)
		expectedHash       = []byte("test_hash")
		expectedBlockEvent = &testBlockEvent{
			Block: comettypes.Block{
				Header: comettypes.Header{
					Height: 1,
					Time:   time.Now(),
					LastBlockID: comettypes.BlockID{
						Hash: expectedHash,
					},
				},
			},
		}
		ctx = context.Background()
	)

	// Set up a mock connection and dialer which are expected to be used once.
	connMock, dialerMock := testeventsquery.NewOneTimeMockConnAndDialer(t)
	connMock.EXPECT().Send(gomock.Any()).Return(nil).Times(1)
	// Mock the Receive method to return the expected block event.
	connMock.EXPECT().Receive().DoAndReturn(func() ([]byte, error) {
		blockEventJson, err := json.Marshal(expectedBlockEvent)
		require.NoError(t, err)

		// Slow the rate at which block events are published to prevent bogging
		// down the test with a really tight async loop which causes test timeout
		// failures to occur.
		time.Sleep(10 * time.Millisecond)
		return blockEventJson, nil
	}).AnyTimes()

	// Set up events query client dependency.
	dialerOpt := eventsquery.WithDialer(dialerMock)
	eventsQueryClient := testeventsquery.NewLocalnetClient(t, dialerOpt)
	deps := depinject.Supply(eventsQueryClient)

	// Set up block client.
	blockClient, err := block.NewBlockClient(ctx, deps, testclient.CometLocalWebsocketURL)
	require.NoError(t, err)
	require.NotNil(t, blockClient)

	// Wait a tick for the observables to be set up. This isn't strictly
	// necessary but is done to mitigate flakiness.
	time.Sleep(10 * time.Millisecond)

	tests := []struct {
		name string
		fn   func() client.Block
	}{
		{
			name: "LatestBlock",
			fn: func() client.Block {
				return blockClient.LatestBlock(ctx)
			},
		},
		{
			name: "CommittedBlocksSequence",
			fn: func() client.Block {
				blockObservable := blockClient.CommittedBlocksSequence(ctx)
				require.NotNil(t, blockObservable)

				// Ensure that the observable is replayable via Last.
				lastBlock := blockObservable.Last(ctx, 1)[0]
				require.Equal(t, expectedHeight, lastBlock.Height())
				require.Equal(t, expectedHash, lastBlock.Hash())

				// Ensure that the observable is replayable via Subscribe.
				blockObserver := blockObservable.Subscribe(ctx)
				for _ = range blockObserver.Ch() {
					// TODO_THIS_COMMIT: should we assert that this matches lastBlock?
					break
				}

				return lastBlock
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var eitherActualBlockCh = make(chan client.Block, 10)

			// Run test functions concurrently because they can block, leading
			// to an unresponsive test. If any of the methods under test hang,
			// the test will time out in the select statement that follows.
			go func(fn func() client.Block) {
				eitherActualBlockCh <- fn()
				close(eitherActualBlockCh)
			}(tt.fn)

			select {
			case actualBlock := <-eitherActualBlockCh:
				require.Equal(t, expectedHeight, actualBlock.Height())
				require.Equal(t, expectedHash, actualBlock.Hash())
			case <-time.After(testTimeoutDuration):
				t.Fatal("timed out waiting for block event")
			}
		})
	}

	blockClient.Close()
}

/*
TODO_TECHDEBT/TODO_CONSIDERATION(#XXX): this duplicates the unexported block event

type from pkg/client/block/block.go. We seem to have some conflicting preferences
which result in the need for this duplication until a preferred direction is
identified:

  - We should prefer tests being in their own pkgs (e.g. block_test)
  - this would resolve if this test were in the block package instead.
  - We should prefer to not export types which don't require exporting for API
    consumption.
  - This test is the only external (to the block pkg) dependency of cometBlockEvent.
  - We could use the //go:build test constraint on a new file which exports it
    for testing purposes.
  - This would imply that we also add -tags=test to all applicable tooling
    and add a test which fails if the tag is absent.
*/
type testBlockEvent struct {
	Block comettypes.Block `json:"block"`
}
