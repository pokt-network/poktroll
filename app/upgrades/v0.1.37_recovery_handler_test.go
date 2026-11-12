package upgrades_test

import (
	"testing"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/keepers"
	"github.com/pokt-network/poktroll/app/pocket"
	"github.com/pokt-network/poktroll/app/upgrades"
	testkeeper "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/sample"
	appkeeper "github.com/pokt-network/poktroll/x/application/keeper"
	gatewaykeeper "github.com/pokt-network/poktroll/x/gateway/keeper"
	sharedkeeper "github.com/pokt-network/poktroll/x/shared/keeper"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// TestUpgrade_0_1_37_Handler_Recovery runs the real handler against a planted supplier
// (first entry of the list, unbonding under another owner). With the recovery address
// set the owner is rewritten; while it is a placeholder the entry is skipped and the
// supplier is left unchanged.
func TestUpgrade_0_1_37_Handler_Recovery(t *testing.T) {
	previousOwner := sample.AccAddressBech32()
	k, ctx := testkeeper.SupplierKeeper(t)
	recoveries := upgrades.SupplierOwnerRecoveries()
	operator, newOwner := recoveries[0].OperatorAddress, recoveries[0].NewOwnerAddress
	stake := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, suppliertypes.DefaultMinStake.Amount.Int64())
	k.SetAndIndexDehydratedSupplier(ctx, sharedtypes.Supplier{
		OwnerAddress:            previousOwner,
		OperatorAddress:         operator,
		Stake:                   &stake,
		UnstakeSessionEndHeight: 100,
	})

	// The address migrations only use the shared keeper, so the other dependencies are nil.
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	sharedKeeper := k.SharedKeeper.(sharedkeeper.Keeper)
	storeService := runtime.NewKVStoreService(k.StoreKey.(*storetypes.KVStoreKey))
	handler := upgrades.Upgrade_0_1_37.CreateUpgradeHandler(nil, &keepers.Keepers{
		SupplierKeeper:    *k.Keeper,
		ServiceKeeper:     k.ServiceKeeper,
		ApplicationKeeper: appkeeper.NewKeeper(cdc, storeService, log.NewNopLogger(), authority, nil, nil, nil, sharedKeeper),
		GatewayKeeper:     gatewaykeeper.NewKeeper(cdc, storeService, log.NewNopLogger(), authority, nil, sharedKeeper),
	}, nil)
	_, err := handler(ctx, upgradetypes.Plan{Name: upgrades.Upgrade_0_1_37_PlanName}, module.VersionMap{})
	require.NoError(t, err)

	supplier, found := k.GetDehydratedSupplier(ctx, operator)
	require.True(t, found)
	if newOwner != "" {
		require.Equal(t, newOwner, supplier.OwnerAddress)
		require.Equal(t, !upgrades.RecoveryCancelsUnbonding, supplier.IsUnbonding())
	} else {
		require.Equal(t, previousOwner, supplier.OwnerAddress, "placeholder recovery address must be skipped")
		require.True(t, supplier.IsUnbonding())
	}
}

// TestUpgrade_0_1_37_Handler_RecoveryFullList plants every listed operator as an
// unbonding supplier under another owner, plus one unlisted supplier, and runs the real
// handler twice under a finite gas meter: every listed owner is rewritten and unbonding
// (one listed supplier is planted staked and begins unbonding), the unlisted one is
// untouched, and a replay of the upgrade block leaves the state unchanged.
func TestUpgrade_0_1_37_Handler_RecoveryFullList(t *testing.T) {
	k, ctx := testkeeper.SupplierKeeper(t)
	previousOwner, unlistedOperator := sample.AccAddressBech32(), sample.AccAddressBech32()
	stake := cosmostypes.NewInt64Coin(pocket.DenomuPOKT, suppliertypes.DefaultMinStake.Amount.Int64())
	recoveries := upgrades.SupplierOwnerRecoveries()
	plant := func(operator string, unstakeSessionEndHeight uint64) {
		k.SetAndIndexDehydratedSupplier(ctx, sharedtypes.Supplier{
			OwnerAddress: previousOwner, OperatorAddress: operator, Stake: &stake, UnstakeSessionEndHeight: unstakeSessionEndHeight,
		})
	}
	// Every listed supplier is unbonding except the second, which is staked.
	for i, r := range recoveries {
		if i == 1 {
			plant(r.OperatorAddress, sharedtypes.SupplierNotUnstaking)
		} else {
			plant(r.OperatorAddress, 100)
		}
	}
	plant(unlistedOperator, sharedtypes.SupplierNotUnstaking)

	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	sharedKeeper := k.SharedKeeper.(sharedkeeper.Keeper)
	storeService := runtime.NewKVStoreService(k.StoreKey.(*storetypes.KVStoreKey))
	handler := upgrades.Upgrade_0_1_37.CreateUpgradeHandler(nil, &keepers.Keepers{
		SupplierKeeper:    *k.Keeper,
		ServiceKeeper:     k.ServiceKeeper,
		ApplicationKeeper: appkeeper.NewKeeper(cdc, storeService, log.NewNopLogger(), authority, nil, nil, nil, sharedKeeper),
		GatewayKeeper:     gatewaykeeper.NewKeeper(cdc, storeService, log.NewNopLogger(), authority, nil, sharedKeeper),
	}, nil)

	const gasLimit = 200_000_000
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx).WithGasMeter(storetypes.NewGasMeter(gasLimit))
	check := func() {
		for i, r := range recoveries {
			supplier, found := k.GetDehydratedSupplier(sdkCtx, r.OperatorAddress)
			require.True(t, found)
			require.Equal(t, r.NewOwnerAddress, supplier.OwnerAddress)
			require.True(t, supplier.IsUnbonding())
			if i != 1 {
				require.EqualValues(t, 100, supplier.UnstakeSessionEndHeight, "an unbonding in progress is kept")
			}
		}
		unlisted, found := k.GetDehydratedSupplier(sdkCtx, unlistedOperator)
		require.True(t, found)
		require.Equal(t, previousOwner, unlisted.OwnerAddress, "unlisted supplier must be untouched")
		require.False(t, unlisted.IsUnbonding())
	}
	for run := 1; run <= 2; run++ {
		_, err := handler(sdkCtx, upgradetypes.Plan{Name: upgrades.Upgrade_0_1_37_PlanName}, module.VersionMap{})
		require.NoError(t, err, "run %d", run)
		check()
	}
	t.Logf("gas consumed by two handler runs over %d suppliers: %d (limit %d)", len(recoveries)+1, sdkCtx.GasMeter().GasConsumed(), gasLimit)
	require.Less(t, sdkCtx.GasMeter().GasConsumed(), uint64(gasLimit))
}
