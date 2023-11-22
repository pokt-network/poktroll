package testblock

import (
	"context"
	"testing"

	"cosmossdk.io/depinject"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/testutil/mockclient"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/block"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/testutil/testclient"
	"github.com/pokt-network/poktroll/testutil/testclient/testeventsquery"
)

// NewLocalnetClient creates and returns a new BlockClient that's configured for
// use with the localnet sequencer.
func NewLocalnetClient(ctx context.Context, t *testing.T) client.BlockClient {
	t.Helper()

	queryClient := testeventsquery.NewLocalnetClient(t)
	require.NotNil(t, queryClient)

	deps := depinject.Supply(queryClient)
	bClient, err := block.NewBlockClient(ctx, deps, testclient.CometLocalWebsocketURL)
	require.NoError(t, err)

	return bClient
}

// NewAnyTimesCommittedBlocksSequenceBlockClient creates a new mock BlockClient.
// This mock BlockClient will expect any number of calls to CommittedBlocksSequence,
// and when that call is made, it returns the given BlocksObservable.
func NewAnyTimesCommittedBlocksSequenceBlockClient(
	t *testing.T,
	blocksObs observable.Observable[client.Block],
) *mockclient.MockBlockClient {
	t.Helper()

	// Create a mock for the block client which expects the LatestBlock method to be called any number of times.
	blockClientMock := NewAnyTimeLatestBlockBlockClient(t, nil, 0)

	// Set up the mock expectation for the CommittedBlocksSequence method. When
	// the method is called, it returns a new replay observable that publishes
	// blocks sent on the given blocksPublishCh.
	blockClientMock.EXPECT().
		CommittedBlocksSequence(
			gomock.AssignableToTypeOf(context.Background()),
		).
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

	// Create a mock for the block client which expects the LatestBlock method to be called any number of times.
	blockClientMock := NewAnyTimeLatestBlockBlockClient(t, nil, 0)

	// Set up the mock expectation for the CommittedBlocksSequence method. When
	// the method is called, it returns a new replay observable that publishes
	// blocks sent on the given blocksPublishCh.
	blockClientMock.EXPECT().CommittedBlocksSequence(
		gomock.AssignableToTypeOf(context.Background()),
	).DoAndReturn(func(ctx context.Context) client.BlocksObservable {
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

// NewAnyTimeLatestBlockBlockClient creates a mock BlockClient that expects
// calls to the LatestBlock method any number of times. When the LatestBlock
// method is called, it returns a mock Block with the provided hash and height.
func NewAnyTimeLatestBlockBlockClient(
	t *testing.T,
	hash []byte,
	height int64,
) *mockclient.MockBlockClient {
	t.Helper()
	ctrl := gomock.NewController(t)

	// Create a mock block that returns the provided hash and height.
	blockMock := NewAnyTimesBlock(t, hash, height)
	// Create a mock block client that expects calls to LatestBlock method and
	// returns the mock block.
	blockClientMock := mockclient.NewMockBlockClient(ctrl)
	blockClientMock.EXPECT().LatestBlock(gomock.Any()).Return(blockMock).AnyTimes()

	return blockClientMock
}

// NewAnyTimesBlock creates a mock Block that expects calls to Height and Hash
// methods any number of times. When the methods are called, they return the
// provided height and hash respectively.
func NewAnyTimesBlock(t *testing.T, hash []byte, height int64) *mockclient.MockBlock {
	t.Helper()
	ctrl := gomock.NewController(t)

	// Create a mock block that returns the provided hash and height AnyTimes.
	blockMock := mockclient.NewMockBlock(ctrl)
	blockMock.EXPECT().Height().Return(height).AnyTimes()
	blockMock.EXPECT().Hash().Return(hash).AnyTimes()

	return blockMock
}
