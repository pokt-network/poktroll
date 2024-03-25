package block_test

import (
	"context"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	"github.com/cometbft/cometbft/libs/json"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	rpctypes "github.com/cometbft/cometbft/rpc/jsonrpc/types"
	"github.com/cometbft/cometbft/types"
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
		expectedHeight = int64(1)
		expectedHash   = []byte("test_hash")

		expectedBlockEvent = &testBlockEvent{
			Data: testBlockEventDataStruct{
				Value: testBlockEventValueStruct{
					Block: &types.Block{
						Header: comettypes.Header{
							Height: 1,
							Time:   time.Now(),
						},
					},
					BlockID: types.BlockID{
						Hash: expectedHash,
					},
				},
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

	cometClient := &testCometClient{
		expectedHeight: expectedHeight,
		expectedHash:   expectedHash,
	}

	// Set up block client.
	blockClient, err := block.NewBlockClient(ctx, cometClient, deps)
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
			name: "LastBlock successfully returns latest block",
			fn: func() client.Block {
				lastBlock := blockClient.LastBlock()
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

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualBlockCh := make(chan client.Block, 10)

			// Run test functions asynchronously because they can block, leading
			// to an unresponsive test. If any of the methods under test hang,
			// the test will time out in the select statement that follows.
			go func(fn func() client.Block) {
				actualBlockCh <- fn()
				close(actualBlockCh)
			}(test.fn)

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

// TODO_BLOCKER: Fix duplicate definitions of this type across tests & source code.
// This duplicates the unexported `cometBlockEvent` from `pkg/client/block/block.go`.
// We need to answer the following questions to avoid this:
//   - Should tests be their own packages? (i.e. `package block` vs `package block_test`)
//   - Should we prefer export types which are not required for API consumption?
//   - Should we use `//go:build“ test constraint on new files using it for testing purposes?
//   - Should we enforce all tests to use `-tags=test`?
type testBlockEvent struct {
	Data testBlockEventDataStruct `json:"data"`
}
type testBlockEventDataStruct struct {
	Value testBlockEventValueStruct `json:"value"`
}
type testBlockEventValueStruct struct {
	Block   *types.Block  `json:"block"`
	BlockID types.BlockID `json:"block_id"`
}

type testCometClient struct {
	expectedHeight int64
	expectedHash   []byte
}

func (t *testCometClient) Block(ctx context.Context, height *int64) (*coretypes.ResultBlock, error) {
	block := &coretypes.ResultBlock{
		Block: &comettypes.Block{},
	}
	block.Block.Height = t.expectedHeight
	block.BlockID.Hash = t.expectedHash

	return block, nil
}
