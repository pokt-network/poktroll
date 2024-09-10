package session_test

import (
	"context"
	"math"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/golang/mock/gomock"
	"github.com/pokt-network/smt"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
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
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	"github.com/pokt-network/poktroll/x/shared"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestRelayerSessionsManager_ColdStartRelayMinerWithUnclaimedRelays(t *testing.T) {
	t.Skip("TODO_TEST: Add a test case which simulates a cold-started relayminer with unclaimed relays.")
}

// requireProofCountEqualsExpectedValueFromProofParams sets up the session manager
// along with its dependencies before starting it.
// It takes in the proofParams to configure the proof requirements and the proofCount
// to assert the number of proofs to be requested.
// TODO_BETA(@red-0ne): Add a test case which verifies that the service's compute units per relay is used as
// the weight of a relay when updating a session's SMT.
func requireProofCountEqualsExpectedValueFromProofParams(t *testing.T, proofParams prooftypes.Params, proofCount int) {
	var (
		_, ctx         = testpolylog.NewLoggerWithCtx(context.Background(), polyzero.DebugLevel)
		spec           = smt.NewTrieSpec(protocol.NewTrieHasher(), true)
		emptyBlockHash = make([]byte, spec.PathHasherSize())
		activeSession  *sessiontypes.Session
		service        sharedtypes.Service
	)

	service = sharedtypes.Service{
		Id:                   "svc",
		ComputeUnitsPerRelay: 2,
	}
	// Add the service to the existing services.
	testqueryclients.AddToExistingServices(t, service)

	activeSession = &sessiontypes.Session{
		Header: &sessiontypes.SessionHeader{
			SessionStartBlockHeight: 1,
			SessionEndBlockHeight:   2,
			Service:                 &service,
		},
	}
	sessionHeader := activeSession.GetHeader()

	// Set up dependencies.
	blocksObs, blockPublishCh := channel.NewReplayObservable[client.Block](ctx, 20)
	blockClient := testblock.NewAnyTimesCommittedBlocksSequenceBlockClient(t, emptyBlockHash, blocksObs)
	supplierOperatorAddress := sample.AccAddress()
	supplierClientMap := testsupplier.NewClaimProofSupplierClientMap(ctx, t, supplierOperatorAddress, proofCount)

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

	serviceQueryClientMock := testqueryclients.NewTestServiceQueryClient(t)
	proofQueryClientMock := testqueryclients.NewTestProofQueryClientWithParams(t, &proofParams)

	deps := depinject.Supply(
		blockClient,
		blockQueryClientMock,
		supplierClientMap,
		sharedQueryClientMock,
		serviceQueryClientMock,
		proofQueryClientMock,
	)
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
	waitSimulateIO()

	// Publish a mined relay to the minedRelaysPublishCh to insert into the session tree.
	minedRelay := testrelayer.NewUnsignedMinedRelay(t, activeSession, supplierOperatorAddress)
	minedRelaysPublishCh <- minedRelay

	// The relayerSessionsManager should have created a session tree for the relay.
	waitSimulateIO()

	// Publish a block to the blockPublishCh to simulate non-actionable blocks.
	sessionStartHeight := sessionHeader.GetSessionStartBlockHeight()
	noopBlock := testblock.NewAnyTimesBlock(t, emptyBlockHash, sessionStartHeight)
	blockPublishCh <- noopBlock

	waitSimulateIO()

	// Calculate the session grace period end block height to emit that block height
	// to the blockPublishCh to trigger claim creation for the session.
	sharedParams := sharedtypes.DefaultParams()
	sessionEndHeight := sessionHeader.GetSessionEndBlockHeight()
	claimWindowOpenHeight := shared.GetClaimWindowOpenHeight(&sharedParams, sessionEndHeight)
	earliestSupplierClaimCommitHeight := shared.GetEarliestSupplierClaimCommitHeight(
		&sharedParams,
		sessionEndHeight,
		emptyBlockHash,
		supplierOperatorAddress,
	)

	claimOpenHeightBlock := testblock.NewAnyTimesBlock(t, emptyBlockHash, claimWindowOpenHeight)
	blockPublishCh <- claimOpenHeightBlock

	waitSimulateIO()

	// Publish a block to the blockPublishCh to trigger claim creation for the session.
	triggerClaimBlock := testblock.NewAnyTimesBlock(t, emptyBlockHash, earliestSupplierClaimCommitHeight)
	blockPublishCh <- triggerClaimBlock

	waitSimulateIO()

	// TODO_IMPROVE: ensure correctness of persisted session trees here.

	proofWindowOpenHeight := shared.GetProofWindowOpenHeight(&sharedParams, sessionEndHeight)
	proofPathSeedBlock := testblock.NewAnyTimesBlock(t, emptyBlockHash, proofWindowOpenHeight)
	blockPublishCh <- proofPathSeedBlock

	waitSimulateIO()

	// Publish a block to the blockPublishCh to trigger proof submission for the session.
	earliestSupplierProofCommitHeight := shared.GetEarliestSupplierProofCommitHeight(
		&sharedParams,
		sessionEndHeight,
		emptyBlockHash,
		supplierOperatorAddress,
	)
	triggerProofBlock := testblock.NewAnyTimesBlock(t, emptyBlockHash, earliestSupplierProofCommitHeight)
	blockPublishCh <- triggerProofBlock

	waitSimulateIO()
}

func TestRelayerSessionsManager_ProofThresholdRequired(t *testing.T) {
	proofParams := prooftypes.DefaultParams()

	// Set proof requirement threshold to a low enough value so a proof is always requested.
	proofParams.ProofRequirementThreshold = 1

	// The test is submitting a single claim. Having the proof requirement threshold
	// set to 1 results in exactly 1 proof being requested.
	numExpectedProofs := 1

	requireProofCountEqualsExpectedValueFromProofParams(t, proofParams, numExpectedProofs)
}

func TestRelayerSessionsManager_ProofProbabilityRequired(t *testing.T) {
	proofParams := prooftypes.DefaultParams()

	// Set proof requirement threshold to max uint64 to skip the threshold check.
	proofParams.ProofRequirementThreshold = math.MaxUint64
	// Set proof request probability to 1 so a proof is always requested.
	proofParams.ProofRequestProbability = 1

	// The test is submitting a single claim. Having the proof request probability
	// set to 1 results in exactly 1 proof being requested.
	numExpectedProofs := 1

	requireProofCountEqualsExpectedValueFromProofParams(t, proofParams, numExpectedProofs)
}

func TestRelayerSessionsManager_ProofNotRequired(t *testing.T) {
	proofParams := prooftypes.DefaultParams()

	// Set proof requirement threshold to max uint64 to skip the threshold check.
	proofParams.ProofRequirementThreshold = math.MaxUint64
	// Set proof request probability to 0 so a proof is never requested.
	proofParams.ProofRequestProbability = 0

	// The test is submitting a single claim. Having the proof request probability
	// set to 0 and proof requirement threshold set to max uint64 results in no proofs
	// being requested.
	numExpectedProofs := 0

	requireProofCountEqualsExpectedValueFromProofParams(t, proofParams, numExpectedProofs)
}

// waitSimulateIO sleeps for a bit to allow the relayer sessions manager to
// process asynchronously. This effectively simulates I/O delays which would
// normally be present.
func waitSimulateIO() {
	time.Sleep(50 * time.Millisecond)
}
