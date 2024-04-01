package session_test

import (
	"context"
	"crypto/sha256"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	"github.com/pokt-network/smt"
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

	// TODO_TECHDEBT: Centralize the configuration for the SMT spec.
	var (
		_, ctx         = testpolylog.NewLoggerWithCtx(context.Background(), polyzero.DebugLevel)
		spec           = smt.NoPrehashSpec(sha256.New(), true)
		emptyBlockHash = make([]byte, spec.PathHasherSize())
	)

	// Set up dependencies.
	blocksObs, blockPublishCh := channel.NewReplayObservable[client.Block](ctx, 1)
	blockClient := testblock.NewAnyTimesCommittedBlocksSequenceBlockClient(t, emptyBlockHash, blocksObs)
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
	noopBlock := testblock.NewAnyTimesBlock(t, emptyBlockHash, sessionStartHeight)
	blockPublishCh <- noopBlock

	// Calculate the session grace period end block height to emit that block height
	// to the blockPublishCh to trigger claim creation for the session.
	sessionGracePeriodEndBlockHeight := int64(sessionEndHeight + sessionkeeper.GetSessionGracePeriodBlockCount())

	// Publish a block to the blockPublishCh to trigger claim creation for the session.
	// TODO_TECHDEBT: assumes claiming at sessionGracePeriodEndBlockHeight is valid.
	// This will likely change in future work.
	triggerClaimBlock := testblock.NewAnyTimesBlock(t, emptyBlockHash, sessionGracePeriodEndBlockHeight)
	blockPublishCh <- triggerClaimBlock

	// TODO_IMPROVE: ensure correctness of persisted session trees here.

	// Publish a block to the blockPublishCh to trigger proof submission for the session.
	// TODO_TECHDEBT: assumes proving at sessionGracePeriodEndBlockHeight + 1 is valid.
	// This will likely change in future work.
	triggerProofBlock := testblock.NewAnyTimesBlock(t, emptyBlockHash, sessionGracePeriodEndBlockHeight+1)
	blockPublishCh <- triggerProofBlock

	// Wait a tick to allow the relayer sessions manager to process asynchronously.
	time.Sleep(250 * time.Millisecond)
}
