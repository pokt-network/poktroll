package session_test

import (
	"context"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/session"
	"github.com/pokt-network/poktroll/testutil/testclient/testblock"
	"github.com/pokt-network/poktroll/testutil/testclient/testsupplier"
	"github.com/pokt-network/poktroll/testutil/testpolylog"
	"github.com/pokt-network/poktroll/testutil/testrelayer"
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
)

func TestRelayerSessionsManager_Start(t *testing.T) {
	const (
		sessionStartHeight = 1
		sessionEndHeight   = 2
	)
	var (
		zeroByteSlice = []byte{0}
		_, ctx        = testpolylog.NewLoggerWithCtx(context.Background(), polyzero.DebugLevel)
	)

	// Set up dependencies.
	blocksObs, blockPublishCh := channel.NewReplayObservable[client.Block](ctx, 1)
	blockClient := testblock.NewAnyTimesCommittedBlocksSequenceBlockClient(t, blocksObs)
	supplierClient := testsupplier.NewOneTimeClaimProofSupplierClient(ctx, t)

	deps := depinject.Supply(blockClient, supplierClient)
	storesDirectoryOpt := testrelayer.WithTempStoresDirectory(t)

	// Create a new relayer sessions manager.
	relayerSessionsManager, err := session.NewRelayerSessions(ctx, deps, storesDirectoryOpt)
	require.NoError(t, err)
	require.NotNil(t, relayerSessionsManager)

	// Pass a mined relays observable via InsertRelays.
	mrObs, minedRelaysPublishCh := channel.NewObservable[*relayer.MinedRelay]()
	minedRelaysObs := relayer.MinedRelaysObservable(mrObs)
	relayerSessionsManager.InsertRelays(minedRelaysObs)

	// Start the relayer sessions manager.
	relayerSessionsManager.Start(ctx)

	// Publish a mined relay to the minedRelaysPublishCh to insert into the session tree.
	minedRelay := testrelayer.NewMinedRelay(t, sessionStartHeight, sessionEndHeight)
	minedRelaysPublishCh <- minedRelay

	// Wait a tick to allow the relayer sessions manager to process asynchronously.
	// It should have created a session tree for the relay.
	time.Sleep(10 * time.Millisecond)

	// Publish a block to the blockPublishCh to simulate non-actionable blocks.
	noopBlock := testblock.NewAnyTimesBlock(t, zeroByteSlice, sessionStartHeight)
	blockPublishCh <- noopBlock

	sessionGracePeriodEndBlockHeight := int64(sessionEndHeight + sessionkeeper.SessionGracePeriod)
	// Publish a block to the blockPublishCh to trigger claim creation for the session.
	// TODO_TECHDEBT: assumes claiming at sessionGracePeriodEndBlockHeight is valid.
	// This will likely change in future work.
	triggerClaimBlock := testblock.NewAnyTimesBlock(t, zeroByteSlice, sessionGracePeriodEndBlockHeight)
	blockPublishCh <- triggerClaimBlock

	// TODO_IMPROVE: ensure correctness of persisted session trees here.

	// Publish a block to the blockPublishCh to trigger proof submission for the session.
	// TODO_TECHDEBT: assumes proving at sessionGracePeriodEndBlockHeight + 1 is valid.
	// This will likely change in future work.
	triggerProofBlock := testblock.NewAnyTimesBlock(t, zeroByteSlice, sessionGracePeriodEndBlockHeight+1)
	blockPublishCh <- triggerProofBlock

	// Wait a tick to allow the relayer sessions manager to process asynchronously.
	time.Sleep(250 * time.Millisecond)
}
