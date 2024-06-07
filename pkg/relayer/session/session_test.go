package session_test

import (
	"context"
	"crypto/sha256"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/golang/mock/gomock"
	"github.com/pokt-network/smt"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/session"
	"github.com/pokt-network/poktroll/testutil/mockclient"
	"github.com/pokt-network/poktroll/testutil/testclient/testblock"
	"github.com/pokt-network/poktroll/testutil/testclient/testqueryclients"
	"github.com/pokt-network/poktroll/testutil/testclient/testsupplier"
	"github.com/pokt-network/poktroll/testutil/testpolylog"
	"github.com/pokt-network/poktroll/testutil/testrelayer"
	"github.com/pokt-network/poktroll/x/shared"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestRelayerSessionsManager_Start(t *testing.T) {
	const (
		sessionStartHeight = 1
		sessionEndHeight   = 2
	)

	// TODO_TECHDEBT(#446): Centralize the configuration for the SMT spec.
	var (
		_, ctx         = testpolylog.NewLoggerWithCtx(context.Background(), polyzero.DebugLevel)
		spec           = smt.NewTrieSpec(sha256.New(), true)
		emptyBlockHash = make([]byte, spec.PathHasherSize())
	)

	// Set up dependencies.
	blocksObs, blockPublishCh := channel.NewReplayObservable[client.Block](ctx, 1)
	blockClient := testblock.NewAnyTimesCommittedBlocksSequenceBlockClient(t, emptyBlockHash, blocksObs)
	supplierClient := testsupplier.NewOneTimeClaimProofSupplierClient(ctx, t)

	ctrl := gomock.NewController(t)
	blockQueryClientMock := mockclient.NewMockCometRPC(ctrl)
	blockQueryClientMock.EXPECT().
		Block(gomock.Any(), gomock.AssignableToTypeOf((*int64)(nil))).
		DoAndReturn(
			func(_ context.Context, height *int64) (*coretypes.ResultBlock, error) {
				// Default to height 1 if none given.
				// See: https://pkg.go.dev/github.com/cometbft/cometbft@v0.38.7/rpc/client#SignClient
				if height == nil {
					height = new(int64)
					*height = 1
				}

				return &coretypes.ResultBlock{
					BlockID: cmttypes.BlockID{
						Hash: []byte("expected block hash"),
					},
					Block: &cmttypes.Block{
						Header: cmttypes.Header{
							Height: *height,
						},
					},
				}, nil
			},
		).
		AnyTimes()

	sharedQueryClientMock := testqueryclients.NewTestSharedQueryClient(t)

	deps := depinject.Supply(blockClient, blockQueryClientMock, supplierClient, sharedQueryClientMock)
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
	//sessionClaimWindowOpenHeight := int64(sessionEndHeight + shared.SessionGracePeriodBlocks)
	sharedParams := sharedtypes.DefaultParams()
	sessionClaimWindowOpenHeight := shared.GetClaimWindowOpenHeight(&sharedParams, sessionEndHeight)

	// Publish a block to the blockPublishCh to trigger claim creation for the session.
	// TODO_BLOCKER(@bryanchriswhite, #516): assumes claiming at sessionClaimWindowOpenHeight is valid.
	// This will likely change in future work.
	// TODO_IN_THIS_PR: Remove -2 after the discussion in GetClaimWindowOpenHeight
	triggerClaimBlock := testblock.NewAnyTimesBlock(t, emptyBlockHash, sessionClaimWindowOpenHeight+2)
	blockPublishCh <- triggerClaimBlock

	// TODO_IMPROVE: ensure correctness of persisted session trees here.

	// Publish a block to the blockPublishCh to trigger proof submission for the session.
	// TODO_BLOCKER(@bryanchriswhite, #516): assumes proving at sessionClaimWindowOpenHeight + 1 is valid.
	// This will likely change in future work.
	// TODO_IN_THIS_PR: Remove -2 after the discussion in GetClaimWindowOpenHeight
	triggerProofBlock := testblock.NewAnyTimesBlock(t, emptyBlockHash, sessionClaimWindowOpenHeight+3)
	blockPublishCh <- triggerProofBlock

	// // Wait a tick to allow the relayer sessions manager to process asynchronously.
	time.Sleep(250 * time.Millisecond)
}
