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

const blockAssertionLoopTimeout = 500 * time.Millisecond

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

	// Run LatestBlock and CommittedBlockSequence concurrently because they can
	// block, leading to an unresponsive test. This function sends multiple values
	// on the actualBlockCh which are all asserted against in blockAssertionLoop.
	// If any of the methods under test hang, the test will time out.
	var (
		actualBlockCh = make(chan client.Block, 1)
		done          = make(chan struct{}, 1)
	)
	go func() {
		// Test LatestBlock method.
		actualBlock := blockClient.LatestBlock(ctx)
		require.Equal(t, expectedHeight, actualBlock.Height())
		require.Equal(t, expectedHash, actualBlock.Hash())

		// Test CommittedBlockSequence method.
		blockObservable := blockClient.CommittedBlocksSequence(ctx)
		require.NotNil(t, blockObservable)

		// Ensure that the observable is replayable via Last.
		actualBlockCh <- blockObservable.Last(ctx, 1)[0]

		// Ensure that the observable is replayable via Subscribe.
		blockObserver := blockObservable.Subscribe(ctx)
		for block := range blockObserver.Ch() {
			actualBlockCh <- block
			break
		}

		// Signal test completion
		done <- struct{}{}
	}()

	// blockAssertionLoop ensures that the blocks retrieved from both LatestBlock
	// method and CommittedBlocksSequence method match the expected block height
	// and hash. This loop waits for blocks to be sent on the actualBlockCh channel
	// by the methods being tested. Once the methods are done, they send a signal on
	// the "done" channel. If the blockAssertionLoop doesn't receive any block or
	// the done signal within a specific timeout, it assumes something has gone wrong
	// and fails the test.
blockAssertionLoop:
	for {
		select {
		case actualBlock := <-actualBlockCh:
			require.Equal(t, expectedHeight, actualBlock.Height())
			require.Equal(t, expectedHash, actualBlock.Hash())
		case <-done:
			break blockAssertionLoop
		case <-time.After(blockAssertionLoopTimeout):
			t.Fatal("timed out waiting for block event")
		}
	}

	// Wait a tick for the observables to be set up.
	time.Sleep(time.Millisecond)

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
