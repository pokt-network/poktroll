package keeper_test

import (
	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/pocket"
	testproof "github.com/pokt-network/poktroll/testutil/proof"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testtree"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	supplierkeeper "github.com/pokt-network/poktroll/x/supplier/keeper"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// After settlement slashes a supplier with a separate owner below min stake, its
// operator can cancel the unbonding by staking min stake again.
func (s *TestSuite) TestSettlePendingClaims_SlashBelowMinStake_OperatorCancelsUnbonding() {
	t := s.T()
	ctx := s.ctx
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx).WithBlockHeight(1)
	sharedParams := s.keepers.SharedKeeper.GetParams(ctx)
	claim := s.claims[0]
	serviceId := claim.GetSessionHeader().GetServiceId()
	operatorAddr := claim.SupplierOperatorAddress
	supplierKeeper := s.keepers.SupplierKeeper.(*supplierkeeper.Keeper)

	// Make the supplier a two-address supplier: owner != operator.
	ownerAddr := sample.AccAddressBech32()
	supplier, found := supplierKeeper.GetDehydratedSupplier(sdkCtx, operatorAddr)
	require.True(t, found)
	supplier.OwnerAddress = ownerAddr
	supplierKeeper.SetDehydratedSupplier(sdkCtx, supplier)

	proofRequirementThreshold, err := claim.GetClaimeduPOKT(sharedParams, s.relayMiningDifficulties[0])
	require.NoError(t, err)
	proofRequirementThreshold = proofRequirementThreshold.Sub(uPOKTCoin(1))
	proofParams := s.keepers.ProofKeeper.GetParams(ctx)
	proofParams.ProofRequestProbability = 0
	proofParams.ProofRequirementThreshold = &proofRequirementThreshold
	proofParams.ProofMissingPenalty = &cosmostypes.Coin{Denom: pocket.DenomuPOKT, Amount: math.NewInt(supplierStakeAmt)}
	require.NoError(t, s.keepers.ProofKeeper.SetParams(ctx, proofParams))

	appStake := cosmostypes.NewCoin("upokt", math.NewInt(1000000))
	appAddr := sample.AccAddressBech32()
	s.keepers.SetApplication(s.ctx, apptypes.Application{
		Address:        appAddr,
		Stake:          &appStake,
		ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{{ServiceId: serviceId}},
	})
	sessionRes, err := s.keepers.GetSession(sdkCtx, &sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: appAddr,
		ServiceId:          serviceId,
		BlockHeight:        1,
	})
	require.NoError(t, err)
	merkleRoot := testproof.SmstRootWithSumAndCount(1000, 1000)
	s.keepers.UpsertClaim(ctx, *testtree.NewClaim(t, operatorAddr, sessionRes.Session.Header, merkleRoot))

	settleHeight := sharedtypes.GetProofWindowCloseHeight(&sharedParams, claim.SessionHeader.SessionEndBlockHeight)
	sdkCtx = sdkCtx.WithBlockHeight(settleHeight)
	_, _, _, err = s.keepers.SettlePendingClaims(sdkCtx)
	require.NoError(t, err)

	slashed, found := supplierKeeper.GetDehydratedSupplier(sdkCtx, operatorAddr)
	require.True(t, found)
	require.True(t, slashed.IsUnbonding())

	// Fund the operator with min stake, then restake as the operator.
	minStake := *supplierKeeper.GetParams(sdkCtx).MinStake
	funds := cosmostypes.NewCoins(cosmostypes.NewCoin(pocket.DenomuPOKT, minStake.Amount.MulRaw(2)))
	require.NoError(t, s.keepers.MintCoins(sdkCtx, tokenomicstypes.ModuleName, funds))
	require.NoError(t, s.keepers.SendCoinsFromModuleToAccount(sdkCtx, tokenomicstypes.ModuleName, cosmostypes.MustAccAddressFromBech32(operatorAddr), funds))

	srv := supplierkeeper.NewMsgServerImpl(*supplierKeeper)
	_, err = srv.StakeSupplier(sdkCtx, &suppliertypes.MsgStakeSupplier{
		Signer:          operatorAddr,
		OwnerAddress:    ownerAddr,
		OperatorAddress: operatorAddr,
		Stake:           &minStake,
		Services: []*sharedtypes.SupplierServiceConfig{{
			ServiceId: serviceId,
			Endpoints: []*sharedtypes.SupplierEndpoint{{Url: "http://localhost:8080", RpcType: sharedtypes.RPCType_JSON_RPC}},
			RevShare:  []*sharedtypes.ServiceRevenueShare{{Address: ownerAddr, RevSharePercentage: 100}},
		}},
	})
	require.NoError(t, err, "operator must be able to rescue a settlement below-min unbonding")

	rescued, found := supplierKeeper.GetDehydratedSupplier(sdkCtx, operatorAddr)
	require.True(t, found)
	require.False(t, rescued.IsUnbonding())
}
