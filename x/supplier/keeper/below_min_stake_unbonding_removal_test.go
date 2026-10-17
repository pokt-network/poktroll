package keeper_test

import (
	"testing"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/pocket"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/supplier/keeper"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// The below min stake unbonding record is removed with the supplier: a new
// supplier at the same operator address that its owner unstakes can only have
// that unbonding cancelled by the owner.
func TestMsgServer_StakeSupplier_BelowMinStakeRecordRemovedWithSupplier(t *testing.T) {
	supplierModuleKeepers, ctx := keepertest.SupplierKeeper(t)
	srv := keeper.NewMsgServerImpl(*supplierModuleKeepers.Keeper)

	ownerAddr := sample.AccAddressBech32()
	operatorAddr := sample.AccAddressBech32()
	minStake := suppliertypes.DefaultMinStake.Amount.Int64()
	stakeMsg, _ := newSupplierStakeMsg(ownerAddr, operatorAddr, minStake, serviceID)
	_, err := srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)

	// Settlement unbonds it below min stake (marker set), then it is removed.
	supplier, found := supplierModuleKeepers.GetDehydratedSupplier(ctx, operatorAddr)
	require.True(t, found)
	slashed := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, minStake-1)
	supplier.Stake = &slashed
	sharedParams := supplierModuleKeepers.SharedKeeper.GetParams(ctx)
	supplierModuleKeepers.BeginSupplierUnbonding(ctx, &supplier, &sharedParams, suppliertypes.SupplierUnbondingReason_SUPPLIER_UNBONDING_REASON_BELOW_MIN_STAKE)
	supplierModuleKeepers.RemoveSupplier(ctx, operatorAddr)

	// A new supplier on the same operator address; the owner unstakes and a slash
	// takes the stake below min stake during unbonding.
	_, err = srv.StakeSupplier(ctx, stakeMsg)
	require.NoError(t, err)
	_, err = srv.UnstakeSupplier(ctx, &suppliertypes.MsgUnstakeSupplier{Signer: ownerAddr, OperatorAddress: operatorAddr})
	require.NoError(t, err)
	supplier, found = supplierModuleKeepers.GetDehydratedSupplier(ctx, operatorAddr)
	require.True(t, found)
	supplier.Stake = &slashed
	supplierModuleKeepers.SetDehydratedSupplier(ctx, supplier)

	operatorRestakeMsg, _ := newSupplierStakeMsg(ownerAddr, operatorAddr, minStake, serviceID)
	setStakeMsgSigner(operatorRestakeMsg, operatorAddr)
	_, err = srv.StakeSupplier(ctx, operatorRestakeMsg)
	require.ErrorContains(t, err, "only the owner can cancel an in-progress unstake")
}
