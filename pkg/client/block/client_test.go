package block_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	comettypes "github.com/cometbft/cometbft/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/block"
	"github.com/pokt-network/poktroll/testutil/testclient/testeventsquery"
)

const (
	testTimeoutDuration = 100 * time.Millisecond

	// duplicates pkg/client/block/client.go's committedBlocksQuery for testing purposes
	committedBlocksQuery = "tm.event='NewBlock'"
)

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

	expectedEventBz, err := json.Marshal(expectedBlockEvent)
	require.NoError(t, err)

	eventsQueryClient := testeventsquery.NewAnyTimesEventsBytesEventsQueryClient(
		ctx, t,
		committedBlocksQuery,
		expectedEventBz,
	)

	deps := depinject.Supply(eventsQueryClient)

	// Set up block client.
	blockClient, err := block.NewBlockClient(ctx, deps)
	require.NoError(t, err)
	require.NotNil(t, blockClient)

	tests := []struct {
		name string
		fn   func() client.Block
	}{
		{
			name: "LastNBlocks(1) successfully returns latest block",
			fn: func() client.Block {
				lastBlock := blockClient.LastNBlocks(ctx, 1)[0]
				return lastBlock
			},
		},
		{
			name: "CommittedBlocksSequence successfully returns latest block",
			fn: func() client.Block {
				blockObservable := blockClient.CommittedBlocksSequence(ctx)
				require.NotNil(t, blockObservable)

				// Ensure that the observable is replayable via Last.
				lastBlock := blockObservable.Last(ctx, 1)[0]
				require.Equal(t, expectedHeight, lastBlock.Height())
				require.Equal(t, expectedHash, lastBlock.Hash())

				return lastBlock
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualBlockCh := make(chan client.Block, 10)

			// Run test functions asynchronously because they can block, leading
			// to an unresponsive test. If any of the methods under test hang,
			// the test will time out in the select statement that follows.
			go func(fn func() client.Block) {
				actualBlockCh <- fn()
				close(actualBlockCh)
			}(tt.fn)

			select {
			case actualBlock := <-actualBlockCh:
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
