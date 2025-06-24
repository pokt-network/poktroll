package block_test

import (
	"context"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cometbft/cometbft/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/block"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/testutil/mockclient"
)

const (
	testTimeoutDuration = 100 * time.Millisecond
)

func TestBlockClient(t *testing.T) {
	var (
		expectedHeight = int64(1)
		expectedHash   = []byte("test_hash")

		ctx = context.Background()
	)

	logger := polylog.Ctx(ctx)
	ctrl := gomock.NewController(t)

	// Set up the CometBFT HTTP client mock
	cometHTTPClientMock := mockclient.NewMockClient(ctrl)

	cometHTTPClientMock.EXPECT().
		Block(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, height *int64) (*coretypes.ResultBlock, error) {
			return &coretypes.ResultBlock{
				Block: &types.Block{
					Header: types.Header{
						Height: expectedHeight,
					},
				},
				BlockID: types.BlockID{
					Hash: expectedHash,
				},
			}, nil
		}).
		AnyTimes()
	cometHTTPClientMock.EXPECT().
		Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(make(chan coretypes.ResultEvent), nil)

	deps := depinject.Supply(cometHTTPClientMock, logger)

	// Set up block client.
	blockClient, err := block.NewBlockClient(ctx, deps)
	require.NoError(t, err)
	require.NotNil(t, blockClient)

	tests := []struct {
		name string
		fn   func() client.Block
	}{
		{
			name: "LastBlock successfully returns latest block",
			fn: func() client.Block {
				lastBlock := blockClient.LastBlock(ctx)
				return lastBlock
			},
		},
		{
			name: "LastBlock successfully returns latest block",
			fn: func() client.Block {
				lastBlock := blockClient.LastBlock(ctx)
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
