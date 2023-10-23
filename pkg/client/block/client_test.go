package block_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	comettypes "github.com/cometbft/cometbft/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"pocket/internal/testclient"
	"pocket/internal/testclient/testeventsquery"
	"pocket/pkg/client"
	"pocket/pkg/client/block"
	eventsquery "pocket/pkg/client/events_query"
)

const blockAssertionLoopTimeout = 100 * time.Millisecond

func main() {
	fmt.Println("HELLOO!!!")
}

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
	connMock, dialerMock := testeventsquery.OneTimeMockConnAndDialer(t)
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
	blockClient, err := block.NewBlockClient(ctx, deps, testclient.CometWebsocketURL)
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

// TODO_TECHDEBT/TODO_CONSIDERATION:
// * we should prefer tests being in their own pkgs (e.g. block_test)
// * we should prefer to not export types which don't require exporting for API consumption
// * the cometBlockEvent isn't and doesn't need to be exported (except for this test)
// * TODO_DISCUSS: we could use the //go:build test constraint on a new file which exports it for testing purposes
//   - This would imply that we also add -tags=test to all applicable tooling and add a test which fails if the tag is absent
type testBlockEvent struct {
	Block comettypes.Block `json:"block"`
}
