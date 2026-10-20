package keeper_test

import (
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	testutilevents "github.com/pokt-network/poktroll/testutil/events"
	testproof "github.com/pokt-network/poktroll/testutil/proof"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// A claim whose relay count times the service compute units per relay exceeds
// uint64 is discarded at settlement, even when the truncated product equals its
// compute units (claims created before the check at claim creation existed).
func (s *TestSuite) TestSettlePendingClaims_ClaimDiscarded_ComputeUnitsProductOverflow() {
	t := s.T()
	ctx := s.ctx
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)
	claim := s.claims[0]

	// 3 * 2^63 = 2^64 + 2^63, which truncates to 2^63 in uint64.
	const numRelays = uint64(1) << 63
	s.keepers.SetService(ctx, sharedtypes.Service{
		Id:                   claim.GetSessionHeader().GetServiceId(),
		ComputeUnitsPerRelay: 3,
		OwnerAddress:         s.keepers.GetAuthority(),
	})
	claim.RootHash = testproof.SmstRootWithSumAndCount(numRelays, numRelays)

	// No proof is required for the claim: the threshold is above its value.
	claimedUpokt, err := claim.GetClaimeduPOKT(sharedParams, s.relayMiningDifficulties[0])
	require.NoError(t, err)
	proofRequirementThreshold := claimedUpokt.Add(uPOKTCoin(1))
	require.NoError(t, s.keepers.ProofKeeper.SetParams(ctx, prooftypes.Params{
		ProofRequestProbability:   0,
		ProofRequirementThreshold: &proofRequirementThreshold,
	}))
	s.keepers.UpsertClaim(ctx, claim)

	blockHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, claim.SessionHeader.SessionEndBlockHeight)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(blockHeight)
	settledResult, expiredResult, numDiscardedFaultyClaims, err := s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)
	require.Equal(t, uint64(0), settledResult.GetNumClaims())
	require.Equal(t, uint64(0), expiredResult.GetNumClaims())
	require.Equal(t, uint64(1), numDiscardedFaultyClaims)

	discardedEvents := testutilevents.FilterEvents[*tokenomicstypes.EventClaimDiscarded](t, sdkCtx.EventManager().Events())
	require.Len(t, discardedEvents, 1)
	require.Contains(t, discardedEvents[0].GetError(), "number of relays")
}
