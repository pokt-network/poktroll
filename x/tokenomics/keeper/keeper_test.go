package keeper_test

import (
	"context"
	"testing"
	"time"

	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/cmd/poktrolld/cmd"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
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

	sdkCtx  sdk.Context
	ctx     context.Context
	keepers keepertest.TokenomicsModuleKeepers
	claim   prooftypes.Claim
	proof   prooftypes.Proof

	numComputeUnits uint64
}

func (s *TestSuite) SetupTest() {
	supplierAddr := sample.AccAddress()
	appAddr := sample.AccAddress()

	s.keepers, s.ctx = keepertest.NewTokenomicsModuleKeepers(s.T())
	s.sdkCtx = sdk.UnwrapSDKContext(s.ctx)

	// Set the suite numComputeUnits to be above the default threshold.
	s.numComputeUnits = prooftypes.DefaultParams().ProofRequirementThreshold + 1

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
		RootHash: smstRootWithSum(s.numComputeUnits),
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

func TestSettleExpiringClaimsSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (s *TestSuite) TestClaimSettlement_ClaimPendingBeforeSettlement() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sdkCtx := sdk.UnwrapSDKContext(ctx)
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
	numClaimsSettled, numClaimsExpired, err := s.keepers.SettlePendingClaims(sdkCtx)
	// Check that no claims were settled
	require.NoError(t, err)
	require.Equal(t, uint64(0), numClaimsSettled)
	require.Equal(t, uint64(0), numClaimsExpired)
	// Validate that the claim still exists
	claims = s.keepers.GetAllClaims(ctx)
	require.Len(t, claims, 1)

	// 2. Settle pending claims just after the session ended.
	// Expectations: Claims should not be settled because the proof window hasn't closed yet.
	blockHeight = shared.GetProofWindowOpenHeight(
		&sharedParams, claim.SessionHeader.SessionEndBlockHeight,
	) + 2 // session ended but proof window is still open
	sdkCtx = sdkCtx.WithBlockHeight(blockHeight)
	numClaimsSettled, numClaimsExpired, err = s.keepers.SettlePendingClaims(sdkCtx)
	// Check that no claims were settled
	require.NoError(t, err)
	require.Equal(t, uint64(0), numClaimsSettled)
	require.Equal(t, uint64(0), numClaimsExpired)
	// Validate that the claim still exists
	claims = s.keepers.GetAllClaims(ctx)
	require.Len(t, claims, 1)
}

func (s *TestSuite) TestClaimSettlement_ClaimExpired_ProofRequiredAndNotProvided_ViaThreshold() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sdkCtx := sdk.UnwrapSDKContext(ctx)
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
	numClaimsSettled, numClaimsExpired, err := s.keepers.SettlePendingClaims(sdkCtx)
	// Check that no claims were settled
	require.NoError(t, err)
	require.Equal(t, uint64(0), numClaimsSettled)
	require.Equal(t, uint64(1), numClaimsExpired)
	// Validate that the claims expired
	claims = s.keepers.GetAllClaims(ctx)
	require.Len(t, claims, 0)

	// Confirm an expiration event was emitted
	events := sdkCtx.EventManager().Events()
	require.Len(t, events, 5) // minting, burning, settling, etc..
	// Validate the expiration event
	expectedEvent, ok := s.getClaimEvent(events, "poktroll.tokenomics.EventClaimExpired").(*tokenomicstypes.EventClaimExpired)
	require.True(t, ok)
	require.Equal(t, s.numComputeUnits, expectedEvent.ComputeUnits)
}

func (s *TestSuite) TestClaimSettlement_ClaimSettled_ProofRequiredAndProvided_ViaThreshold() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sdkCtx := sdk.UnwrapSDKContext(ctx)
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
	numClaimsSettled, numClaimsExpired, err := s.keepers.SettlePendingClaims(sdkCtx)
	// Check that no claims were settled
	require.NoError(t, err)
	require.Equal(t, uint64(1), numClaimsSettled)
	require.Equal(t, uint64(0), numClaimsExpired)
	// Validate that the claims expired
	claims = s.keepers.GetAllClaims(ctx)
	require.Len(t, claims, 0)

	// Confirm an settlement event was emitted
	events := sdkCtx.EventManager().Events()
	expectedEvent, ok := s.getClaimEvent(events, "poktroll.tokenomics.EventClaimSettled").(*tokenomicstypes.EventClaimSettled)
	require.True(t, ok)
	require.True(t, expectedEvent.ProofRequired)
	require.Equal(t, s.numComputeUnits, expectedEvent.ComputeUnits)
}

func (s *TestSuite) TestClaimSettlement_ClaimSettled_ProofRequiredAndProvided_ViaProbability() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)

	// Set the proof parameters to require a proof via probability AND NOT via threshold.
	err := s.keepers.ProofKeeper.SetParams(ctx, prooftypes.Params{
		ProofRequestProbability:   1,
		ProofRequirementThreshold: 9001,
	})
	require.NoError(t, err)

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
	// NB: proof window has definitely closed at this point
	blockHeight := shared.GetProofWindowCloseHeight(&sharedParams, claim.SessionHeader.SessionEndBlockHeight)
	sdkCtx = sdkCtx.WithBlockHeight(blockHeight)
	numClaimsSettled, numClaimsExpired, err := s.keepers.SettlePendingClaims(sdkCtx)
	// Check that no claims were settled
	require.NoError(t, err)
	require.Equal(t, uint64(1), numClaimsSettled)
	require.Equal(t, uint64(0), numClaimsExpired)
	// Validate that the claims expired
	claims = s.keepers.GetAllClaims(ctx)
	require.Len(t, claims, 0)

	// Confirm an settlement event was emitted
	events := sdkCtx.EventManager().Events()
	expectedEvent, ok := s.getClaimEvent(events, "poktroll.tokenomics.EventClaimSettled").(*tokenomicstypes.EventClaimSettled)
	require.True(t, ok)
	require.True(t, expectedEvent.ProofRequired)
	require.Equal(t, s.numComputeUnits, expectedEvent.ComputeUnits)
}

func (s *TestSuite) TestClaimSettlement_Settles_WhenAProofIsNotRequired() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)

	// Create a claim that does not require a proof
	claim := s.claim

	// Set the proof parameters to NOT require via probability NOR threshold.
	err := s.keepers.ProofKeeper.SetParams(ctx, prooftypes.Params{
		ProofRequestProbability:   0,
		ProofRequirementThreshold: s.numComputeUnits + 1,
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
	numClaimsSettled, numClaimsExpired, err := s.keepers.SettlePendingClaims(sdkCtx)
	// Check that no claims were settled
	require.NoError(t, err)
	require.Equal(t, uint64(1), numClaimsSettled)
	require.Equal(t, uint64(0), numClaimsExpired)
	// Validate that the claims expired
	claims = s.keepers.GetAllClaims(ctx)
	require.Len(t, claims, 0)

	// Confirm an expiration event was emitted
	events := sdkCtx.EventManager().Events()
	expectedEvent, ok := s.getClaimEvent(events, "poktroll.tokenomics.EventClaimSettled").(*tokenomicstypes.EventClaimSettled)
	require.True(t, ok)
	require.False(t, expectedEvent.ProofRequired)
	require.Equal(t, s.numComputeUnits, expectedEvent.ComputeUnits)
}

func (s *TestSuite) TestClaimSettlement_DoesNotSettle_BeforeProofWindowCloses() {
	s.T().Skip("TODO_TEST: Implement that a claim remains unsettled before the proof window closes")
}

func (s *TestSuite) TestClaimSettlement_DoesNotSettle_IfProofIsInvalid() {
	s.T().Skip("TODO_TEST: Implement that a claim remains unsettled before the proof window closes")
}

func (s *TestSuite) TestClaimSettlement_DoesNotSettle_IfProofIsRequiredButMissing() {
	s.T().Skip("TODO_TEST: Implement that a claim remains unsettled before the proof window closes")
}

func (s *TestSuite) TestClaimSettlement_MultipleClaimsSettle_WithMultipleApplicationsAndSuppliers() {
	s.T().Skip("TODO_TEST: Implement that multiple claims settle at once when different sessions have overlapping applications and suppliers")
}

// getClaimEvent verifies that there is exactly one event of type protoType in
// the given events and returns it. If there are 0 or more than 1 events of the
// given type, it fails the test.
func (s *TestSuite) getClaimEvent(events sdk.Events, protoType string) proto.Message {
	var parsedEvent proto.Message
	numExpectedEvents := 0
	for _, event := range events {
		switch event.Type {
		case protoType:
			var err error
			parsedEvent, err = sdk.ParseTypedEvent(abci.Event(event))
			s.Require().NoError(err)
			numExpectedEvents++
		default:
			continue
		}
	}
	if numExpectedEvents == 1 {
		return parsedEvent
	}
	require.NotEqual(s.T(), 1, numExpectedEvents, "Expected exactly one claim event")
	return nil
}
