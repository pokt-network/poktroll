// Package testblock provides utilities and mock clients to facilitate testing
// interactions with blockchain-related functionality. It includes tools for creating mock
// BlockClients, Block observables, and mock blocks tailored for specific testing scenarios.
// The package is designed to help ensure that tests around blockchain functionality are
// robust, using mock implementations to replicate expected behaviors in controlled environments.
//
// Given its role in testing, the testblock package leverages several other testing
// packages and libraries, such as gomock, testify, and internal testing clients
// from the pocket project.
package testblock

import (
	"context"
	"testing"

	"cosmossdk.io/depinject"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"pocket/internal/mocks/mockclient"
	"pocket/internal/testclient"
	"pocket/internal/testclient/testeventsquery"
	"pocket/pkg/client"
	"pocket/pkg/client/block"
	"pocket/pkg/observable/channel"
)

// NewLocalnetClient creates and returns a new BlockClient for localnet testing
// environments.
//
// Parameters:
// - ctx: The context for creating the client.
// - t: The testing.T instance for assertions.
//
// The function initializes a localnet query client, ensures its successful creation, and then
// proceeds to set up the dependencies required to instantiate a new BlockClient.
// The final BlockClient instance connects to the CometLocalWebsocketURL endpoint.
//
// Returns:
// - A new instance of client.BlockClient configured for localnet interactions.
func NewLocalnetClient(ctx context.Context, t *testing.T) client.BlockClient {
	t.Helper()

	queryClient := testeventsquery.NewLocalnetClient(t)
	require.NotNil(t, queryClient)

	deps := depinject.Supply(queryClient)
	bClient, err := block.NewBlockClient(ctx, deps, testclient.CometLocalWebsocketURL)
	require.NoError(t, err)

	return bClient
}

// NewOneTimeCommittedBlocksSequenceBlockClient creates a new mock BlockClient.
// This mock BlockClient will expect a call to CommittedBlocksSequence, and
// when that call is made, it returns a new BlocksObservable that is notified of
// blocks sent on the given blocksPublishCh.
//
// Parameters:
// - t: *testing.T - The test instance.
// - blocksPublishCh: chan client.Block - The channel from which blocks are published to the observable.
//
// Returns:
// - *mockclient.MockBlockClient: The mock block client.
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
//
// Parameters:
// - t: *testing.T - The test instance.
// - hash: []byte - The expected hash value that the mock Block should return.
// - height: int64 - The expected block height that the mock Block should return.
//
// Returns:
// - *mockclient.MockBlockClient: The mock block client.
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
//
// Parameters:
// - t: *testing.T - The test instance.
// - hash: []byte - The expected hash value that the mock Block should return.
// - height: int64 - The expected block height that the mock Block should return.
//
// Returns:
// - *mockclient.MockBlock: The mock block.
func NewAnyTimesBlock(t *testing.T, hash []byte, height int64) *mockclient.MockBlock {
	t.Helper()
	ctrl := gomock.NewController(t)

	// Create a mock block that returns the provided hash and height AnyTimes.
	blockMock := mockclient.NewMockBlock(ctrl)
	blockMock.EXPECT().Height().Return(height).AnyTimes()
	blockMock.EXPECT().Hash().Return(hash).AnyTimes()

	return blockMock
}
