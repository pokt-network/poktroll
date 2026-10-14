package keeper_test

import (
	"fmt"
	"testing"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/pocket"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/supplier/keeper"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// stakeNSuppliers stakes n suppliers at min stake with 2 services each, cap disabled.
func stakeNSuppliers(t *testing.T, n int) (keepertest.SupplierModuleKeepers, cosmostypes.Context, []string) {
	t.Helper()
	k, ctx := keepertest.SupplierKeeper(t)
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	operators := make([]string, 0, n)
	for i := 0; i < n; i++ {
		ownerAddr, operatorAddr := sample.AccAddressBech32(), sample.AccAddressBech32()
		msg := multiServiceStakeMsg(t, k, sdkCtx, ownerAddr, operatorAddr, capMinStake, fmt.Sprintf("a%d", i), fmt.Sprintf("b%d", i))
		_, err := srv.StakeSupplier(ctx, msg)
		require.NoError(t, err)
		operators = append(operators, operatorAddr)
	}
	return k, sdkCtx, operators
}

func updatePrice(t *testing.T, k keepertest.SupplierModuleKeepers, ctx cosmostypes.Context, price int64) {
	t.Helper()
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	coin := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, price)
	_, err := srv.UpdateParam(ctx, &suppliertypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      suppliertypes.ParamStakePerAdditionalService,
		AsType:    &suppliertypes.MsgUpdateParam_AsCoin{AsCoin: &coin},
	})
	require.NoError(t, err)
}

func TestServiceCapSweep_ScheduledByCapParamUpdateOnly(t *testing.T) {
	k, ctx, _ := stakeNSuppliers(t, 1)
	require.False(t, k.IsServiceCapSweepPending(ctx))

	// staking_fee is not a cap param.
	fee := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 2)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	_, err := srv.UpdateParam(ctx, &suppliertypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      suppliertypes.ParamStakingFee,
		AsType:    &suppliertypes.MsgUpdateParam_AsCoin{AsCoin: &fee},
	})
	require.NoError(t, err)
	require.False(t, k.IsServiceCapSweepPending(ctx))

	updatePrice(t, k, ctx, capPrice)
	require.True(t, k.IsServiceCapSweepPending(ctx))
}

func TestServiceCapSweep_BulkUpdateSchedulesOnlyOnCapChange(t *testing.T) {
	k, ctx, _ := stakeNSuppliers(t, 1)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	updatePrice(t, k, ctx, capPrice)
	for k.IsServiceCapSweepPending(ctx) {
		k.EndBlockerServiceCapSweep(ctx)
	}
	params := k.Keeper.GetParams(ctx)

	fee := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, 3)
	params.StakingFee = &fee
	_, err := srv.UpdateParams(ctx, &suppliertypes.MsgUpdateParams{Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(), Params: params})
	require.NoError(t, err)
	require.False(t, k.IsServiceCapSweepPending(ctx))

	params.MinStakeIncludedServices = 5
	_, err = srv.UpdateParams(ctx, &suppliertypes.MsgUpdateParams{Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(), Params: params})
	require.NoError(t, err)
	require.True(t, k.IsServiceCapSweepPending(ctx))
}

func TestServiceCapSweep_ProcessesInBatchesAndClears(t *testing.T) {
	numSuppliers := keeper.ServiceCapSweepBatchSize + 5
	k, ctx, operators := stakeNSuppliers(t, numSuppliers)
	updatePrice(t, k, ctx, capPrice) // min stake now covers 1 of 2 services

	require.Equal(t, keeper.ServiceCapSweepBatchSize, k.EndBlockerServiceCapSweep(ctx))
	require.True(t, k.IsServiceCapSweepPending(ctx))
	require.Equal(t, 5, k.EndBlockerServiceCapSweep(ctx))
	require.False(t, k.IsServiceCapSweepPending(ctx))
	require.Equal(t, 0, k.EndBlockerServiceCapSweep(ctx))

	for _, operatorAddr := range operators {
		require.Len(t, countedServiceIds(t, k, ctx, operatorAddr), 1, operatorAddr)
	}
}

func TestServiceCapSweep_ParamChangeMidSweepRestarts(t *testing.T) {
	numSuppliers := keeper.ServiceCapSweepBatchSize + 5
	k, ctx, operators := stakeNSuppliers(t, numSuppliers)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	setIncluded := func(included uint64) {
		_, err := srv.UpdateParam(ctx, &suppliertypes.MsgUpdateParam{
			Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
			Name:      suppliertypes.ParamMinStakeIncludedServices,
			AsType:    &suppliertypes.MsgUpdateParam_AsUint64{AsUint64: included},
		})
		require.NoError(t, err)
	}

	// Loose cap first: min stake covers both services, so the first batch changes nothing.
	setIncluded(2)
	updatePrice(t, k, ctx, capPrice)
	require.Equal(t, keeper.ServiceCapSweepBatchSize, k.EndBlockerServiceCapSweep(ctx))
	for _, operatorAddr := range operators {
		require.Len(t, countedServiceIds(t, k, ctx, operatorAddr), 2)
	}

	// Tighten mid-sweep: the cursor resets, so the suppliers already visited are
	// visited again with the new value.
	setIncluded(1)
	processed := 0
	for k.IsServiceCapSweepPending(ctx) {
		processed += k.EndBlockerServiceCapSweep(ctx)
	}
	require.Equal(t, numSuppliers, processed)
	for _, operatorAddr := range operators {
		require.Len(t, countedServiceIds(t, k, ctx, operatorAddr), 1, operatorAddr)
	}
}

func TestServiceCapSweep_DisabledCapClearsWithoutWork(t *testing.T) {
	k, ctx, operators := stakeNSuppliers(t, 3)
	updatePrice(t, k, ctx, capPrice)
	updatePrice(t, k, ctx, 0) // disabled again before the sweep ran
	require.Equal(t, 0, k.EndBlockerServiceCapSweep(ctx))
	require.False(t, k.IsServiceCapSweepPending(ctx))
	for _, operatorAddr := range operators {
		require.Len(t, countedServiceIds(t, k, ctx, operatorAddr), 2)
	}
}

func TestServiceCapSweep_SkipsZeroServiceAndUnbondingSuppliers(t *testing.T) {
	k, ctx, _ := stakeNSuppliers(t, 1)
	srv := keeper.NewMsgServerImpl(*k.Keeper)
	ownerAddr, zeroServicesOperator := sample.AccAddressBech32(), sample.AccAddressBech32()
	_, err := srv.StakeSupplier(ctx, &suppliertypes.MsgStakeSupplier{
		Signer: ownerAddr, OwnerAddress: ownerAddr, OperatorAddress: zeroServicesOperator,
		Stake: &cosmostypes.Coin{Denom: pocket.DenomuPOKT, Amount: math.NewInt(capMinStake)},
	})
	require.NoError(t, err)

	updatePrice(t, k, ctx, capPrice)
	for k.IsServiceCapSweepPending(ctx) {
		k.EndBlockerServiceCapSweep(ctx)
	}
	stored, found := k.GetDehydratedSupplier(ctx, zeroServicesOperator)
	require.True(t, found)
	require.False(t, stored.IsUnbonding())
}
