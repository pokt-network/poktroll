package session_test

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	sdkmath "cosmossdk.io/math"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	cmttypes "github.com/cometbft/cometbft/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/smt"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/supplier"
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
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestRelayerSessionsManager_ColdStartRelayMinerWithUnclaimedRelays(t *testing.T) {
	t.Skip("TODO_TEST: Add a test case which simulates a cold-started relayminer with unclaimed relays.")
}

// requireProofCountEqualsExpectedValueFromProofParams sets up the session manager
// along with its dependencies before starting it.
// It takes in the proofParams to configure the proof requirements and the proofCount
// to assert the number of proofs to be requested.
// TODO_MAINNET(@red-0ne): Add a test case which verifies that the service's compute units per relay is used as
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

	testqueryclients.SetServiceRelayDifficultyTargetHash(t, service.Id, protocol.BaseRelayDifficultyHashBz)
	// Add the service to the existing services.
	testqueryclients.AddToExistingServices(t, service)

	activeSession = &sessiontypes.Session{
		Header: &sessiontypes.SessionHeader{
			SessionStartBlockHeight: 1,
			SessionEndBlockHeight:   2,
			ServiceId:               service.Id,
			SessionId:               "sessionId",
		},
	}
	supplierOperatorAddress := sample.AccAddress()
	// Set the supplier operator balance to be able to submit the expected number of proofs.
	feePerProof := prooftypes.DefaultParams().ProofSubmissionFee.Amount.Int64()
	gasCost := session.ClamAndProofGasCost.Amount.Int64()
	proofCost := feePerProof + gasCost
	supplierOperatorBalance := proofCost
	supplierClientMap := testsupplier.NewClaimProofSupplierClientMap(ctx, t, supplierOperatorAddress, proofCount)
	blockPublishCh, minedRelaysPublishCh := setupDependencies(t, ctx, supplierClientMap, emptyBlockHash, proofParams, supplierOperatorBalance)

	// Publish a mined relay to the minedRelaysPublishCh to insert into the session tree.
	minedRelay := testrelayer.NewUnsignedMinedRelay(t, activeSession, supplierOperatorAddress)
	minedRelaysPublishCh <- minedRelay

	// The relayerSessionsManager should have created a session tree for the relay.
	waitSimulateIO()

	// Publish a block to the blockPublishCh to simulate non-actionable blocks.
	sessionStartHeight := activeSession.GetHeader().GetSessionStartBlockHeight()
	sessionEndHeight := activeSession.GetHeader().GetSessionEndBlockHeight()

	playClaimAndProofSubmissionBlocks(t, sessionStartHeight, sessionEndHeight, supplierOperatorAddress, emptyBlockHash, blockPublishCh)
}

func TestRelayerSessionsManager_ProofThresholdRequired(t *testing.T) {
	proofParams := prooftypes.DefaultParams()

	// Set proof requirement threshold to a low enough value so a proof is always requested.
	proofParams.ProofRequirementThreshold = uPOKTCoin(1)

	// The test is submitting a single claim. Having the proof requirement threshold
	// set to 1 results in exactly 1 proof being requested.
	numExpectedProofs := 1

	requireProofCountEqualsExpectedValueFromProofParams(t, proofParams, numExpectedProofs)
}

func TestRelayerSessionsManager_ProofProbabilityRequired(t *testing.T) {
	proofParams := prooftypes.DefaultParams()

	// Set proof requirement threshold to max int64 to skip the threshold check.
	proofParams.ProofRequirementThreshold = uPOKTCoin(math.MaxInt64)
	// Set proof request probability to 1 so a proof is always requested.
	proofParams.ProofRequestProbability = 1

	// The test is submitting a single claim. Having the proof request probability
	// set to 1 results in exactly 1 proof being requested.
	numExpectedProofs := 1

	requireProofCountEqualsExpectedValueFromProofParams(t, proofParams, numExpectedProofs)
}

func TestRelayerSessionsManager_ProofNotRequired(t *testing.T) {
	proofParams := prooftypes.DefaultParams()

	// Set proof requirement threshold to max int64 to skip the threshold check.
	proofParams.ProofRequirementThreshold = uPOKTCoin(math.MaxInt64)
	// Set proof request probability to 0 so a proof is never requested.
	proofParams.ProofRequestProbability = 0

	// The test is submitting a single claim. Having the proof request probability
	// set to 0 and proof requirement threshold set to max uint64 results in no proofs
	// being requested.
	numExpectedProofs := 0

	requireProofCountEqualsExpectedValueFromProofParams(t, proofParams, numExpectedProofs)
}

func TestRelayerSessionsManager_InsufficientBalanceForProofSubmission(t *testing.T) {
	var (
		_, ctx         = testpolylog.NewLoggerWithCtx(context.Background(), polyzero.DebugLevel)
		spec           = smt.NewTrieSpec(protocol.NewTrieHasher(), true)
		emptyBlockHash = make([]byte, spec.PathHasherSize())
	)

	proofParams := prooftypes.DefaultParams()

	// Set proof requirement threshold to a low enough value so a proof is always requested.
	proofParams.ProofRequirementThreshold = uPOKTCoin(1)

	// * Add 2 services with different CUPRs
	// * Create 2 claims with the same number of mined relays, each claim for a different service.
	// * Assert that only the claim of the highest CUPR service get its proof submitted.

	lowCUPRService := sharedtypes.Service{
		Id:                   "lowCUPRService",
		ComputeUnitsPerRelay: 1,
	}
	testqueryclients.AddToExistingServices(t, lowCUPRService)
	testqueryclients.SetServiceRelayDifficultyTargetHash(t, lowCUPRService.Id, protocol.BaseRelayDifficultyHashBz)

	highCUPRService := sharedtypes.Service{
		Id:                   "highCUPRService",
		ComputeUnitsPerRelay: 2,
	}
	testqueryclients.AddToExistingServices(t, highCUPRService)
	testqueryclients.SetServiceRelayDifficultyTargetHash(t, highCUPRService.Id, protocol.BaseRelayDifficultyHashBz)

	lowCUPRServiceActiveSession := &sessiontypes.Session{
		Header: &sessiontypes.SessionHeader{
			SessionStartBlockHeight: 1,
			SessionEndBlockHeight:   2,
			ServiceId:               lowCUPRService.Id,
			SessionId:               fmt.Sprintf("%sSessionId", lowCUPRService.Id),
		},
	}

	highCUPRServiceActiveSession := &sessiontypes.Session{
		Header: &sessiontypes.SessionHeader{
			SessionStartBlockHeight: 1,
			SessionEndBlockHeight:   2,
			ServiceId:               highCUPRService.Id,
			SessionId:               fmt.Sprintf("%sSessionId", highCUPRService.Id),
		},
	}

	// Assert that the session start and end block heights are the same for both
	// services and use their common start and end block heights for the test.

	require.Equal(t,
		lowCUPRServiceActiveSession.GetHeader().GetSessionStartBlockHeight(),
		highCUPRServiceActiveSession.GetHeader().GetSessionStartBlockHeight(),
	)
	sessionStartHeight := lowCUPRServiceActiveSession.GetHeader().GetSessionStartBlockHeight()

	require.Equal(t,
		lowCUPRServiceActiveSession.GetHeader().GetSessionEndBlockHeight(),
		highCUPRServiceActiveSession.GetHeader().GetSessionEndBlockHeight(),
	)
	sessionEndHeight := lowCUPRServiceActiveSession.GetHeader().GetSessionEndBlockHeight()

	// Create a supplier client map that expects exactly 1 claim and 1 proof submission
	// even though 2 claims are created.
	ctrl := gomock.NewController(t)
	supplierClientMock := mockclient.NewMockSupplierClient(ctrl)

	supplierOperatorAddress := sample.AccAddress()
	supplierOperatorAccAddress := sdktypes.MustAccAddressFromBech32(supplierOperatorAddress)

	proofSubmissionFee := prooftypes.DefaultParams().ProofSubmissionFee.Amount.Int64()
	claimAndProofGasCost := session.ClamAndProofGasCost.Amount.Int64()
	// Set the supplier operator balance to be able to submit only a single proof.
	supplierOperatorBalance := proofSubmissionFee + claimAndProofGasCost + 1
	supplierClientMock.EXPECT().
		OperatorAddress().
		Return(&supplierOperatorAccAddress).
		AnyTimes()

	supplierClientMock.EXPECT().
		CreateClaims(
			gomock.Eq(ctx),
			gomock.Any(),
			gomock.AssignableToTypeOf(([]client.MsgCreateClaim)(nil)),
		).
		DoAndReturn(func(ctx context.Context, timeoutHeight int64, claimMsgs ...*prooftypes.MsgCreateClaim) error {
			// Assert that only the claim of the highest CUPR service is created.
			require.Len(t, claimMsgs, 1)
			require.Equal(t, claimMsgs[0].SessionHeader.ServiceId, highCUPRService.Id)
			return nil
		}).
		Times(1)

	supplierClientMock.EXPECT().
		SubmitProofs(
			gomock.Eq(ctx),
			gomock.Any(),
			gomock.AssignableToTypeOf(([]client.MsgSubmitProof)(nil)),
		).
		DoAndReturn(func(ctx context.Context, timeoutHeight int64, proofMsgs ...*prooftypes.MsgSubmitProof) error {
			// Assert that only the proof of the highest CUPR service is created.
			require.Len(t, proofMsgs, 1)
			require.Equal(t, proofMsgs[0].SessionHeader.ServiceId, highCUPRService.Id)
			return nil
		}).
		Times(1)

	supplierClientMap := supplier.NewSupplierClientMap()
	supplierClientMap.SupplierClients[supplierOperatorAddress] = supplierClientMock

	blockPublishCh, minedRelaysPublishCh := setupDependencies(t, ctx, supplierClientMap, emptyBlockHash, proofParams, supplierOperatorBalance)

	// For each service, publish a mined relay to the minedRelaysPublishCh to
	// insert into the session tree.
	lowCUPRMinedRelay := testrelayer.NewUnsignedMinedRelay(t, lowCUPRServiceActiveSession, supplierOperatorAddress)
	minedRelaysPublishCh <- lowCUPRMinedRelay

	// The relayerSessionsManager should have created a session tree for the low CUPR relay.
	waitSimulateIO()

	highCUPRMinedRelay := testrelayer.NewUnsignedMinedRelay(t, highCUPRServiceActiveSession, supplierOperatorAddress)
	minedRelaysPublishCh <- highCUPRMinedRelay

	// The relayerSessionsManager should have created a session tree for the high CUPR relay.
	waitSimulateIO()

	playClaimAndProofSubmissionBlocks(t, sessionStartHeight, sessionEndHeight, supplierOperatorAddress, emptyBlockHash, blockPublishCh)
}

// waitSimulateIO sleeps for a bit to allow the relayer sessions manager to
// process asynchronously. This effectively simulates I/O delays which would
// normally be present.
func waitSimulateIO() {
	time.Sleep(50 * time.Millisecond)
}

// uPOKTCoin returns a pointer to a uPOKT denomination coin with the given amount.
func uPOKTCoin(amount int64) *sdktypes.Coin {
	return &sdktypes.Coin{Denom: volatile.DenomuPOKT, Amount: sdkmath.NewInt(amount)}
}

func setupDependencies(
	t *testing.T,
	ctx context.Context,
	supplierClientMap *supplier.SupplierClientMap,
	blockHash []byte,
	proofParams prooftypes.Params,
	supplierOperatorBalance int64,
) (chan<- client.Block, chan<- *relayer.MinedRelay) {
	// Set up dependencies.
	blocksObs, blockPublishCh := channel.NewReplayObservable[client.Block](ctx, 20)
	blockClient := testblock.NewAnyTimesCommittedBlocksSequenceBlockClient(t, blockHash, blocksObs)

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
	bankQueryClient := testqueryclients.NewTestBankQueryClientWithBalance(t, supplierOperatorBalance)

	deps := depinject.Supply(
		blockClient,
		blockQueryClientMock,
		supplierClientMap,
		sharedQueryClientMock,
		serviceQueryClientMock,
		proofQueryClientMock,
		bankQueryClient,
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

	return blockPublishCh, minedRelaysPublishCh
}

// playClaimAndProofSubmissionBlocks simulates the block heights at which claims and proofs
// are submitted by the supplier. It publishes blocks to the blockPublishCh to trigger
// claims and proofs creation for the session number.
func playClaimAndProofSubmissionBlocks(
	t *testing.T,
	sessionStartHeight, sessionEndHeight int64,
	supplierOperatorAddress string,
	blockHash []byte,
	blockPublishCh chan<- client.Block,
) {
	// Publish a block to the blockPublishCh to simulate non-actionable blocks.
	// NB: This only needs to be done once per block regardless of the number of
	// services, claims and proofs.
	noopBlock := testblock.NewAnyTimesBlock(t, blockHash, sessionStartHeight)
	blockPublishCh <- noopBlock

	waitSimulateIO()

	// Calculate the session grace period end block height to emit that block height
	// to the blockPublishCh to trigger session trees processing for the session number.
	sharedParams := sharedtypes.DefaultParams()
	claimWindowOpenHeight := sharedtypes.GetClaimWindowOpenHeight(&sharedParams, sessionEndHeight)
	earliestSupplierClaimCommitHeight := sharedtypes.GetEarliestSupplierClaimCommitHeight(
		&sharedParams,
		sessionEndHeight,
		blockHash,
		supplierOperatorAddress,
	)

	claimOpenHeightBlock := testblock.NewAnyTimesBlock(t, blockHash, claimWindowOpenHeight)
	blockPublishCh <- claimOpenHeightBlock

	waitSimulateIO()

	// Publish a block to the blockPublishCh to trigger claims creation for the session number.
	triggerClaimBlock := testblock.NewAnyTimesBlock(t, blockHash, earliestSupplierClaimCommitHeight)
	blockPublishCh <- triggerClaimBlock

	waitSimulateIO()

	proofWindowOpenHeight := sharedtypes.GetProofWindowOpenHeight(&sharedParams, sessionEndHeight)
	proofPathSeedBlock := testblock.NewAnyTimesBlock(t, blockHash, proofWindowOpenHeight)
	blockPublishCh <- proofPathSeedBlock

	waitSimulateIO()

	// Publish a block to the blockPublishCh to trigger proof submission for the session.
	earliestSupplierProofCommitHeight := sharedtypes.GetEarliestSupplierProofCommitHeight(
		&sharedParams,
		sessionEndHeight,
		blockHash,
		supplierOperatorAddress,
	)
	triggerProofBlock := testblock.NewAnyTimesBlock(t, blockHash, earliestSupplierProofCommitHeight)
	blockPublishCh <- triggerProofBlock

	waitSimulateIO()
}
