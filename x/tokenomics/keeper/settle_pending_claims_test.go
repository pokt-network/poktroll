package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	tokenomicskeeper "github.com/pokt-network/poktroll/x/tokenomics/keeper"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// TODO_IN_THIS_PR_TEST: If some sessions settle, but halfway through it fails, need to make sure it is atomic and the state is reveresed.

func TestSettleExpiringClaimsSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

func (s *TestSuite) TestSettlePendingClaims_ClaimPendingBeforeSettlement() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sdkCtx := sdk.UnwrapSDKContext(ctx)

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
	// Check that no claims were settled
	require.NoError(t, err)
	require.Equal(t, uint64(0), numClaimsSettled)
	require.Equal(t, uint64(0), numClaimsExpired)
	// Validate that the claim still exists
	claims = s.keepers.GetAllClaims(ctx)
	require.Len(t, claims, 1)

	// 2. Settle pending claims just after the session ended.
	// Expectations: Claims should not be settled because the proof window hasn't closed yet.
	// TODO_IMPROVE(@red-0ne, @Olshansk): Use the governance parameters for more
	// precise block heights once they are implemented.
	blockHeight = claim.SessionHeader.SessionEndBlockHeight + 2 // session ended but proof window is still open
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

func (s *TestSuite) TestSettlePendingClaims_ClaimExpired_ProofRequiredAndNotProvided() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Create a claim that requires a proof
	claim := s.claim
	numComputeUnits := uint64(tokenomicskeeper.ProofRequiredComputeUnits + 1)
	claim.RootHash = smstRootWithSum(numComputeUnits)

	// 0. Add the claim & verify it exists
	s.keepers.UpsertClaim(ctx, claim)
	claims := s.keepers.GetAllClaims(ctx)
	s.Require().Len(claims, 1)

	// 1. Settle pending claims after proof window closes
	// Expectation: All (1) claims should be expired.
	// TODO_IMPROVE(@red-0ne, @Olshansk): Use the governance parameters for more precise block heights once they are implemented.
	blockHeight := claim.SessionHeader.SessionEndBlockHeight * 10 // proof window has definitely closed at this point
	sdkCtx = sdkCtx.WithBlockHeight(blockHeight)
	numClaimsSettled, numClaimsExpired, _, err := s.keepers.SettlePendingClaims(sdkCtx)
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
	require.Equal(t, numComputeUnits, expectedEvent.ComputeUnits)
}

func (s *TestSuite) TestSettlePendingClaims_ClaimSettled_ProofRequiredAndProvided() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Create a claim that requires a proof
	claim := s.claim
	numComputeUnits := uint64(tokenomicskeeper.ProofRequiredComputeUnits + 1)
	claim.RootHash = smstRootWithSum(numComputeUnits)

	// 0. Add the claim & verify it exists
	s.keepers.UpsertClaim(ctx, claim)
	claims := s.keepers.GetAllClaims(ctx)
	s.Require().Len(claims, 1)

	// Upsert the proof
	s.keepers.UpsertProof(ctx, s.proof)

	// 1. Settle pending claims after proof window closes
	// Expectation: All (1) claims should be claimed.
	// TODO_IMPROVE(@red-0ne, @Olshansk): Use the governance parameters for more precise block heights once they are implemented.
	blockHeight := s.claim.SessionHeader.SessionEndBlockHeight * 10 // proof window has definitely closed at this point
	sdkCtx = sdkCtx.WithBlockHeight(blockHeight)
	numClaimsSettled, numClaimsExpired, _, err := s.keepers.SettlePendingClaims(sdkCtx)
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
	require.Equal(t, numComputeUnits, expectedEvent.ComputeUnits)

}

func (s *TestSuite) TestSettlePendingClaims_Settles_WhenAProofIsNotRequired() {
	// Retrieve default values
	t := s.T()
	ctx := s.ctx
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Create a claim that does not require a proof
	claim := s.claim
	numComputeUnits := uint64(tokenomicskeeper.ProofRequiredComputeUnits - 1)
	claim.RootHash = smstRootWithSum(numComputeUnits)

	// 0. Add the claim & verify it exists
	s.keepers.UpsertClaim(ctx, claim)
	claims := s.keepers.GetAllClaims(ctx)
	s.Require().Len(claims, 1)

	// 1. Settle pending claims after proof window closes
	// Expectation: All (1) claims should be claimed.
	// TODO_IMPROVE(@red-0ne, @Olshansk): Use the governance parameters for more precise block heights once they are implemented.
	blockHeight := claim.SessionHeader.SessionEndBlockHeight * 10 // proof window has definitely closed at this point
	sdkCtx = sdkCtx.WithBlockHeight(blockHeight)
	numClaimsSettled, numClaimsExpired, _, err := s.keepers.SettlePendingClaims(sdkCtx)
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
	require.Equal(t, numComputeUnits, expectedEvent.ComputeUnits)
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
