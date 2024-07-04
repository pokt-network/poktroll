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
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testclient/testblock"
	"github.com/pokt-network/poktroll/testutil/testclient/testqueryclients"
	"github.com/pokt-network/poktroll/testutil/testclient/testsupplier"
	"github.com/pokt-network/poktroll/testutil/testpolylog"
	"github.com/pokt-network/poktroll/testutil/testrelayer"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	"github.com/pokt-network/poktroll/x/shared"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// TODO_TEST: Add a test case which simulates a cold-started relayminer with unclaimed relays.

func TestRelayerSessionsManager_Start(t *testing.T) {
	// TODO_TECHDEBT(#446): Centralize the configuration for the SMT spec.
	var (
		_, ctx         = testpolylog.NewLoggerWithCtx(context.Background(), polyzero.DebugLevel)
		spec           = smt.NewTrieSpec(sha256.New(), true)
		emptyBlockHash = make([]byte, spec.PathHasherSize())
		activeSession  *sessiontypes.Session
	)

	activeSession = &sessiontypes.Session{
		Header: &sessiontypes.SessionHeader{
			SessionStartBlockHeight: 1,
			SessionEndBlockHeight:   2,
		},
	}
	sessionHeader := activeSession.GetHeader()

	// Set up dependencies.
	blocksObs, blockPublishCh := channel.NewReplayObservable[client.Block](ctx, 20)
	blockClient := testblock.NewAnyTimesCommittedBlocksSequenceBlockClient(t, emptyBlockHash, blocksObs)
	supplierAddress := sample.AccAddress()
	supplierClientMap := testsupplier.NewOneTimeClaimProofSupplierClientMap(ctx, t, supplierAddress)

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

	deps := depinject.Supply(blockClient, blockQueryClientMock, supplierClientMap, sharedQueryClientMock)
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

	// Wait a tick to allow the relayer sessions manager to start.
	time.Sleep(50 * time.Millisecond)

	// Publish a mined relay to the minedRelaysPublishCh to insert into the session tree.
	minedRelay := testrelayer.NewUnsignedMinedRelay(t, activeSession, supplierAddress)
	minedRelaysPublishCh <- minedRelay

	// Wait a tick to allow the relayer sessions manager to process asynchronously.
	// It should have created a session tree for the relay.
	time.Sleep(50 * time.Millisecond)

	// Publish a block to the blockPublishCh to simulate non-actionable blocks.
	sessionStartHeight := sessionHeader.GetSessionStartBlockHeight()
	noopBlock := testblock.NewAnyTimesBlock(t, emptyBlockHash, sessionStartHeight)
	blockPublishCh <- noopBlock

	// Wait a tick to allow the relayer sessions manager to process asynchronously.
	time.Sleep(50 * time.Millisecond)

	// Calculate the session grace period end block height to emit that block height
	// to the blockPublishCh to trigger claim creation for the session.
	sharedParams := sharedtypes.DefaultParams()
	sessionEndHeight := sessionHeader.GetSessionEndBlockHeight()
	claimWindowOpenHeight := shared.GetClaimWindowOpenHeight(&sharedParams, sessionEndHeight)
	earliestSupplierClaimCommitHeight := shared.GetEarliestSupplierClaimCommitHeight(
		&sharedParams,
		sessionEndHeight,
		emptyBlockHash,
		supplierAddress,
	)

	claimOpenHeightBlock := testblock.NewAnyTimesBlock(t, emptyBlockHash, claimWindowOpenHeight)
	blockPublishCh <- claimOpenHeightBlock

	// Wait a tick to allow the relayer sessions manager to process asynchronously.
	time.Sleep(50 * time.Millisecond)

	// Publish a block to the blockPublishCh to trigger claim creation for the session.
	triggerClaimBlock := testblock.NewAnyTimesBlock(t, emptyBlockHash, earliestSupplierClaimCommitHeight)
	blockPublishCh <- triggerClaimBlock

	// Wait a tick to allow the relayer sessions manager to process asynchronously.
	time.Sleep(50 * time.Millisecond)

	// TODO_IMPROVE: ensure correctness of persisted session trees here.

	proofWindowOpenHeight := shared.GetProofWindowOpenHeight(&sharedParams, sessionEndHeight)
	proofPathSeedBlock := testblock.NewAnyTimesBlock(t, emptyBlockHash, proofWindowOpenHeight)
	blockPublishCh <- proofPathSeedBlock

	// Wait a tick to allow the relayer sessions manager to process asynchronously.
	time.Sleep(50 * time.Millisecond)

	// Publish a block to the blockPublishCh to trigger proof submission for the session.
	earliestSupplierProofCommitHeight := shared.GetEarliestSupplierProofCommitHeight(
		&sharedParams,
		sessionEndHeight,
		emptyBlockHash,
		supplierAddress,
	)
	triggerProofBlock := testblock.NewAnyTimesBlock(t, emptyBlockHash, earliestSupplierProofCommitHeight)
	blockPublishCh <- triggerProofBlock

	// Wait a tick to allow the relayer sessions manager to process asynchronously.
	time.Sleep(100 * time.Millisecond)
}
