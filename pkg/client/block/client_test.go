package block_test

import (
	"context"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	"github.com/cometbft/cometbft/libs/json"
	rpctypes "github.com/cometbft/cometbft/rpc/jsonrpc/types"
	"github.com/cometbft/cometbft/types"
	comettypes "github.com/cometbft/cometbft/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/block"
	"github.com/pokt-network/poktroll/testutil/testclient/testeventsquery"
)

const (
	testTimeoutDuration = 100000 * time.Millisecond

	// duplicates pkg/client/block/client.go's committedBlocksQuery for testing purposes
	committedBlocksQuery = "tm.event='NewBlock'"
)

func TestBlockClient(t *testing.T) {
	var (
		expectedHeight     = int64(1)
		expectedHash       = []byte("test_hash")
		expectedBlockEvent = &types.EventDataNewBlock{
			Block: &types.Block{
				Header: comettypes.Header{
					Height: 1,
					Time:   time.Now(),
				},
			},
			BlockID: comettypes.BlockID{
				Hash: expectedHash,
			},
		}
		ctx = context.Background()
	)

	expectedEventBz, err := json.Marshal(expectedBlockEvent)
	require.NoError(t, err)

	expectedRPCResponse := &rpctypes.RPCResponse{
		Result: expectedEventBz,
	}

	expectedRPCResponseBz, err := json.Marshal(expectedRPCResponse)
	require.NoError(t, err)

	eventsQueryClient := testeventsquery.NewAnyTimesEventsBytesEventsQueryClient(
		ctx, t,
		committedBlocksQuery,
		expectedRPCResponseBz,
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
