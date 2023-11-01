//go:build integration

package block_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"pocket/internal/testclient/testblock"
	"pocket/pkg/client"
)

const blockIntegrationSubTimeout = 5 * time.Second

func TestBlockClient_LatestBlock(t *testing.T) {
	ctx := context.Background()

	blockClient := testblock.NewLocalnetClient(ctx, t)
	require.NotNil(t, blockClient)

	block := blockClient.LatestBlock(ctx)
	require.NotEmpty(t, block)
}

func TestBlockClient_BlocksObservable(t *testing.T) {
	ctx := context.Background()

	blockClient := testblock.NewLocalnetClient(ctx, t)
	require.NotNil(t, blockClient)

	blockSub := blockClient.CommittedBlocksSequence(ctx).Subscribe(ctx)

	var (
		blockMu      sync.Mutex
		blockCounter int
		blocksToRecv = 2
		errCh        = make(chan error, 1)
	)
	go func() {
		var previousBlock client.Block
		for block := range blockSub.Ch() {
			if previousBlock != nil {
				if !assert.Equal(t, previousBlock.Height()+1, block.Height()) {
					errCh <- fmt.Errorf("expected block height %d, got %d", previousBlock.Height()+1, block.Height())
					return
				}
			}
			previousBlock = block

			require.NotEmpty(t, block)
			blockMu.Lock()
			blockCounter++
			if blockCounter >= blocksToRecv {
				errCh <- nil
				return
			}
			blockMu.Unlock()
		}
	}()

	select {
	case err := <-errCh:
		require.NoError(t, err)
		require.Equal(t, blocksToRecv, blockCounter)
	case <-time.After(blockIntegrationSubTimeout):
		t.Fatalf(
			"timed out waiting for block subscription; expected %d blocks, got %d",
			blocksToRecv, blockCounter,
		)
	}
}
