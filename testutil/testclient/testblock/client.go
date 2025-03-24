package testblock

import (
	"context"
	"testing"

	"cosmossdk.io/depinject"
	"github.com/cometbft/cometbft/types"
	sdkclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/pokt-network/pocket/pkg/client"
	"github.com/pokt-network/pocket/pkg/client/block"
	"github.com/pokt-network/pocket/pkg/observable"
	"github.com/pokt-network/pocket/pkg/observable/channel"
	"github.com/pokt-network/pocket/testutil/mockclient"
	"github.com/pokt-network/pocket/testutil/testclient"
	"github.com/pokt-network/pocket/testutil/testclient/testeventsquery"
)

// NewLocalnetClient creates and returns a new BlockClient that's configured for
// use with the LocalNet validator.
func NewLocalnetClient(ctx context.Context, t *testing.T) client.BlockClient {
	t.Helper()

	queryClient := testeventsquery.NewLocalnetClient(t)
	require.NotNil(t, queryClient)

	cometClient, err := sdkclient.NewClientFromNode(testclient.CometLocalTCPURL)
	require.NoError(t, err)

	deps := depinject.Supply(queryClient, cometClient)
	bClient, err := block.NewBlockClient(ctx, deps)
	require.NoError(t, err)

	return bClient
}

// NewAnyTimesCommittedBlocksSequenceBlockClient creates a new mock BlockClient.
// This mock BlockClient will expect any number of calls to CommittedBlocksSequence,
// and when that call is made, it returns the given EventsObservable[Block].
func NewAnyTimesCommittedBlocksSequenceBlockClient(
	t *testing.T,
	blockHash []byte,
	blocksObs observable.Observable[client.Block],
) *mockclient.MockBlockClient {
	t.Helper()

	// Create a mock for the block client which expects the LastBlock method to be called any number of times.
	blockClientMock := NewAnyTimeLastBlockBlockClient(t, blockHash, 0)

	// Set up the mock expectation for the CommittedBlocksSequence method. When
	// the method is called, it returns a new replay observable that publishes
	// blocks sent on the given blocksPublishCh.
	blockClientMock.EXPECT().
		CommittedBlocksSequence(gomock.Any()).
		Return(blocksObs).
		AnyTimes()

	return blockClientMock
}

// NewOneTimeCommittedBlocksSequenceBlockClient creates a new mock BlockClient.
// This mock BlockClient will expect a call to CommittedBlocksSequence, and
// when that call is made, it returns a new BlocksObservable that is notified of
// blocks sent on the given blocksPublishCh.
// blocksPublishCh is the channel the caller can use to publish blocks the observable.
func NewOneTimeCommittedBlocksSequenceBlockClient(
	t *testing.T,
	blocksPublishCh chan client.Block,
) *mockclient.MockBlockClient {
	t.Helper()

	// Create a mock for the block client which expects the LastBlock method to be called any number of times.
	blockClientMock := NewAnyTimeLastBlockBlockClient(t, nil, 0)

	// Set up the mock expectation for the CommittedBlocksSequence method. When
	// the method is called, it returns a new replay observable that publishes
	// blocks sent on the given blocksPublishCh.
	blockClientMock.EXPECT().CommittedBlocksSequence(
		gomock.AssignableToTypeOf(context.Background()),
	).DoAndReturn(func(ctx context.Context) client.BlockReplayObservable {
		// Create a new replay observable with a replay buffer size of 1. Blocks
		// are published to this observable via the provided blocksPublishCh.
		withPublisherOpt := channel.WithPublisher(blocksPublishCh)
		obs, _ := channel.NewReplayObservable[client.Block](
			ctx, 1, withPublisherOpt,
		)
		return obs
	})

	return blockClientMock
}

// NewAnyTimeLastBlockBlockClient creates a mock BlockClient that expects
// calls to the LastBlock method any number of times. When the LastBlock
// method is called, it returns a mock Block with the provided hash and height.
func NewAnyTimeLastBlockBlockClient(
	t *testing.T,
	blockHash []byte,
	blockHeight int64,
) *mockclient.MockBlockClient {
	t.Helper()
	ctrl := gomock.NewController(t)

	// Create a mock block that returns the provided hash and height.
	blockMock := NewAnyTimesBlock(t, blockHash, blockHeight)
	// Create a mock block client that expects calls to LastBlock method and
	// returns the mock block.
	blockClientMock := mockclient.NewMockBlockClient(ctrl)
	blockClientMock.EXPECT().LastBlock(gomock.Any()).Return(blockMock).AnyTimes()

	return blockClientMock
}

// NewAnyTimesBlock creates a mock Block that expects calls to Height and Hash
// methods any number of times. When the methods are called, they return the
// provided height and hash respectively.
func NewAnyTimesBlock(
	t *testing.T,
	blockHash []byte,
	blockHeight int64,
) *mockclient.MockBlock {
	t.Helper()
	ctrl := gomock.NewController(t)

	// Create a mock block that returns the provided hash and height AnyTimes.
	blockMock := mockclient.NewMockBlock(ctrl)
	blockMock.EXPECT().Height().Return(blockHeight).AnyTimes()
	blockMock.EXPECT().Hash().Return(blockHash).AnyTimes()
	blockMock.EXPECT().Txs().Return([]types.Tx{}).AnyTimes()

	return blockMock
}
