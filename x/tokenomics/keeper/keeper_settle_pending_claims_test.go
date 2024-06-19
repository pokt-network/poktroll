package keeper_test

import (
	"context"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/cmd/poktrolld/cmd"
	testutilevents "github.com/pokt-network/poktroll/testutil/events"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	testutilproof "github.com/pokt-network/poktroll/testutil/proof"
	"github.com/pokt-network/poktroll/testutil/sample"
	testsession "github.com/pokt-network/poktroll/testutil/session"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	"github.com/pokt-network/poktroll/x/shared"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

const (
	testServiceId = "svc1"
	testSessionId = "mock_session_id"
)
const minExecutionPeriod = 5 * time.Second

func init() {
	cmd.InitSDKConfig()
}

type TestSuite struct {
	suite.Suite

	sdkCtx  cosmostypes.Context
	ctx     context.Context
	keepers keepertest.TokenomicsModuleKeepers
	claim   prooftypes.Claim
	proof   prooftypes.Proof

	expectedComputeUnits uint64
}

// SetupTest creates the following and stores them in the suite:
// - An cosmostypes.Context.
// - A keepertest.TokenomicsModuleKeepers to provide access to integrated keepers.
// - An expectedComputeUnits which is the default proof_requirement_threshold.
// - A claim that will require a proof via threshold, given the default proof params.
// - A proof which contains only the session header supplier address.
func (s *TestSuite) SetupTest() {
	supplierAddr := sample.AccAddress()
	appAddr := sample.AccAddress()

	s.keepers, s.ctx = keepertest.NewTokenomicsModuleKeepers(s.T())
	s.sdkCtx = cosmostypes.UnwrapSDKContext(s.ctx)

	// Set the suite expectedComputeUnits to equal the default proof_requirement_threshold
	// such that by default, s.claim will require a proof 100% of the time.
	s.expectedComputeUnits = prooftypes.DefaultProofRequirementThreshold

	// Prepare a claim that can be inserted
	s.claim = prooftypes.Claim{
		SupplierAddress: supplierAddr,
		SessionHeader: &sessiontypes.SessionHeader{
			ApplicationAddress:      appAddr,
			Service:                 &sharedtypes.Service{Id: testServiceId},
			SessionId:               "session_id",
			SessionStartBlockHeight: 1,
			SessionEndBlockHeight:   testsession.GetSessionEndHeightWithDefaultParams(1),
		},

		// Set the suite expectedComputeUnits to be equal to the default threshold.
		// This SHOULD make the claim require a proof given the default proof parameters.
		RootHash: testutilproof.SmstRootWithSum(s.expectedComputeUnits),
	}

	// Prepare a claim that can be inserted
	s.proof = prooftypes.Proof{
		SupplierAddress: s.claim.SupplierAddress,
		SessionHeader:   s.claim.SessionHeader,
		// ClosestMerkleProof
	}

	appStake := types.NewCoin("upokt", math.NewInt(1000000))
	app := apptypes.Application{
		Address: appAddr,
		Stake:   &appStake,
	}
	s.keepers.SetApplication(s.ctx, app)
}

// TestSettleExpiringClaimsSuite tests the claim settlement process.
// NB: Each test scenario (method) is run in isolation and #TestSetup() is called
// for each prior to running.
func TestSettlePendingClaims(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (s *TestSuite) TestSettlePendingClaims_ClaimPendingBeforeSettlement() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)

	// 0. Add the claim & verify it exists
	claim := s.claim
	s.keepers.UpsertClaim(ctx, claim)
	claims := s.keepers.GetAllClaims(ctx)
	s.Require().Len(claims, 1)

	// 1. Settle pending claims while the session is still active.
	// Expectations: No claims should be settled because the session is still ongoing
	blockHeight := claim.SessionHeader.SessionEndBlockHeight - 2 // session is still active
	sdkCtx = sdkCtx.WithBlockHeight(blockHeight)
	numClaimsSettled, numClaimsExpired, _, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Check that no claims were settled.
	require.Equal(t, uint64(0), numClaimsSettled)

	// Validate that no claims expired.
	require.Equal(t, uint64(0), numClaimsExpired)

	// Validate that one claim still remains.
	claims = s.keepers.GetAllClaims(ctx)
	require.Len(t, claims, 1)

	// Calculate a block height which is within the proof window.
	proofWindowOpenHeight := shared.GetProofWindowOpenHeight(
		&sharedParams, claim.SessionHeader.SessionEndBlockHeight,
	)
	proofWindowCloseHeight := shared.GetProofWindowCloseHeight(
		&sharedParams, claim.SessionHeader.SessionEndBlockHeight,
	)
	blockHeight = (proofWindowCloseHeight - proofWindowOpenHeight) / 2

	// 2. Settle pending claims just after the session ended.
	// Expectations: Claims should not be settled because the proof window hasn't closed yet.
	sdkCtx = sdkCtx.WithBlockHeight(blockHeight)
	numClaimsSettled, numClaimsExpired, _, err = s.keepers.SettlePendingClaims(sdkCtx)
	// Check that no claims were settled
	require.NoError(t, err)
	require.Equal(t, uint64(0), numClaimsSettled)
	require.Equal(t, uint64(0), numClaimsExpired)
	// Validate that the claim still exists
	claims = s.keepers.GetAllClaims(ctx)
	require.Len(t, claims, 1)
}

func (s *TestSuite) TestSettlePendingClaims_ClaimExpired_ProofRequiredAndNotProvided_ViaThreshold() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)

	// Create a claim that requires a proof
	claim := s.claim

	// 0. Add the claim & verify it exists
	s.keepers.UpsertClaim(ctx, claim)
	claims := s.keepers.GetAllClaims(ctx)
	s.Require().Len(claims, 1)

	// 1. Settle pending claims after proof window closes
	// Expectation: All (1) claims should be expired.
	// NB: proofs should be rejected when the current height equals the proof window close height.
	blockHeight := shared.GetProofWindowCloseHeight(&sharedParams, claim.SessionHeader.SessionEndBlockHeight)
	sdkCtx = sdkCtx.WithBlockHeight(blockHeight)
	numClaimsSettled, numClaimsExpired, _, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Check that no claims were settled.
	require.Equal(t, uint64(0), numClaimsSettled)

	// Validate that one claims expired
	require.Equal(t, uint64(1), numClaimsExpired)

	// Validate that no claims remain.
	claims = s.keepers.GetAllClaims(ctx)
	require.Len(t, claims, 0)

	// Confirm an expiration event was emitted
	events := sdkCtx.EventManager().Events()
	require.Len(t, events, 5) // minting, burning, settling, etc..

	expectedEvents := testutilevents.FilterEvents[*tokenomicstypes.EventClaimExpired](t, events, "poktroll.tokenomics.EventClaimExpired")
	require.Len(t, expectedEvents, 1)
	expectedEvent := expectedEvents[0]
	require.Equal(t, s.expectedComputeUnits, expectedEvent.ComputeUnits)
}

func (s *TestSuite) TestSettlePendingClaims_ClaimSettled_ProofRequiredAndProvided_ViaThreshold() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)

	// Create a claim that requires a proof
	claim := s.claim

	// 0. Add the claim & verify it exists
	s.keepers.UpsertClaim(ctx, claim)
	claims := s.keepers.GetAllClaims(ctx)
	s.Require().Len(claims, 1)

	// Upsert the proof
	s.keepers.UpsertProof(ctx, s.proof)

	// 1. Settle pending claims after proof window closes
	// Expectation: All (1) claims should be claimed.
	// NB: proofs should be rejected when the current height equals the proof window close height.
	blockHeight := shared.GetProofWindowCloseHeight(&sharedParams, claim.SessionHeader.SessionEndBlockHeight)
	sdkCtx = sdkCtx.WithBlockHeight(blockHeight)
	numClaimsSettled, numClaimsExpired, _, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Check that one claim was settled.
	require.Equal(t, uint64(1), numClaimsSettled)

	// Validate that no claims expired.
	require.Equal(t, uint64(0), numClaimsExpired)

	// Validate that no claims remain.
	claims = s.keepers.GetAllClaims(ctx)
	require.Len(t, claims, 0)

	// Confirm an settlement event was emitted
	events := sdkCtx.EventManager().Events()
	expectedEvents := testutilevents.FilterEvents[*tokenomicstypes.EventClaimSettled](t, events, "poktroll.tokenomics.EventClaimSettled")
	require.Len(t, expectedEvents, 1)

	expectedEvent := expectedEvents[0]
	require.True(t, expectedEvent.ProofRequired)
	require.Equal(t, s.expectedComputeUnits, expectedEvent.ComputeUnits)
}

func (s *TestSuite) TestSettlePendingClaims_Settles_WhenAProofIsNotRequired() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)

	// Create a claim that does not require a proof
	claim := s.claim

	// Set the proof parameters such that s.claim DOES NOT require a proof because
	// the proof_request_probability is 0% AND because the proof_requirement_threshold
	// exceeds s.expectedComputeUnits, which matches s.claim.
	err := s.keepers.ProofKeeper.SetParams(ctx, prooftypes.Params{
		ProofRequestProbability: 0,
		// +1 to push the threshold above s.claim's compute units
		ProofRequirementThreshold: s.expectedComputeUnits + 1,
	})
	require.NoError(t, err)

	// 0. Add the claim & verify it exists
	s.keepers.UpsertClaim(ctx, claim)
	claims := s.keepers.GetAllClaims(ctx)
	s.Require().Len(claims, 1)

	// 1. Settle pending claims after proof window closes
	// Expectation: All (1) claims should be claimed.
	// NB: proofs should be rejected when the current height equals the proof window close height.
	blockHeight := shared.GetProofWindowCloseHeight(&sharedParams, claim.SessionHeader.SessionEndBlockHeight)
	sdkCtx = sdkCtx.WithBlockHeight(blockHeight)
	numClaimsSettled, numClaimsExpired, _, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	// Check that one claim was settled.
	require.Equal(t, uint64(1), numClaimsSettled)

	// Validate that no claims expired.
	require.Equal(t, uint64(0), numClaimsExpired)

	// Validate that no claims remain.
	claims = s.keepers.GetAllClaims(ctx)
	require.Len(t, claims, 0)

	// Confirm an expiration event was emitted
	events := sdkCtx.EventManager().Events()
	expectedEvents := testutilevents.FilterEvents[*tokenomicstypes.EventClaimSettled](t, events, "poktroll.tokenomics.EventClaimSettled")
	require.Len(t, expectedEvents, 1)
	expectedEvent := expectedEvents[0]
	require.False(t, expectedEvent.ProofRequired)
	require.Equal(t, s.expectedComputeUnits, expectedEvent.ComputeUnits)
}

func (s *TestSuite) TestSettlePendingClaims_DoesNotSettle_BeforeProofWindowCloses() {
	s.T().Skip("TODO_TEST: Implement that a claim remains unsettled before the proof window closes")
}

func (s *TestSuite) TestSettlePendingClaims_DoesNotSettle_IfProofIsInvalid() {
	s.T().Skip("TODO_TEST: Implement that a claim remains unsettled before the proof window closes")
}

func (s *TestSuite) TestSettlePendingClaims_DoesNotSettle_IfProofIsRequiredButMissing() {
	s.T().Skip("TODO_TEST: Implement that a claim remains unsettled before the proof window closes")
}

func (s *TestSuite) TestSettlePendingClaims_MultipleClaimsSettle_WithMultipleApplicationsAndSuppliers() {
	s.T().Skip("TODO_TEST: Implement that multiple claims settle at once when different sessions have overlapping applications and suppliers")
}
