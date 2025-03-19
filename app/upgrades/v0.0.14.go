package upgrades

import (
	"context"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	cosmosTypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/pokt-network/poktroll/app/keepers"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

const Upgrade_0_0_14_PlanName = "v0.0.14"

// Upgrade_0_0_14 handles the upgrade to release `v0.0.14`.
// This is planned to be issued on both Pocket Network's Shannon Alpha & Beta TestNets.
var Upgrade_0_0_14 = Upgrade{
	PlanName: Upgrade_0_0_14_PlanName,
	// No state changes in this upgrade.
	CreateUpgradeHandler: func(mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			logger := cosmosTypes.UnwrapSDKContext(ctx).Logger()
			logger.Info("Starting upgrade handler", "upgrade_plan_name", Upgrade_0_0_14_PlanName)

			supplierKeeper := keepers.SupplierKeeper
			storeAdapter := runtime.KVStoreAdapter(runtime.NewKVStoreService(storetypes.NewKVStoreKey(types.StoreKey)).OpenKVStore(ctx))
			store := prefix.NewStore(storeAdapter, types.KeyPrefix(types.SupplierKeyOperatorPrefix))
			iterator := storetypes.KVStorePrefixIterator(store, []byte{})

			defer iterator.Close()

			for ; iterator.Valid(); iterator.Next() {
				var supplierDeprecated sharedtypes.SupplierDeprecated
				err := supplierDeprecated.Unmarshal(iterator.Value())
				if err != nil {
					logger.Error("Failed to unmarshal supplier data", "upgrade_plan_name", Upgrade_0_0_14_PlanName, "error", err)
					panic(err)
				}

				supplier := sharedtypes.Supplier{
					OperatorAddress:         supplierDeprecated.OperatorAddress,
					Services:                supplierDeprecated.Services,
					OwnerAddress:            supplierDeprecated.OwnerAddress,
					Stake:                   supplierDeprecated.Stake,
					UnstakeSessionEndHeight: supplierDeprecated.UnstakeSessionEndHeight,
					ServiceConfigHistory: []*sharedtypes.ServiceConfigUpdate{
						{
							Services:             supplierDeprecated.Services,
							EffectiveBlockHeight: 1,
						},
					},
				}

				// Update the supplier with the migrated data
				supplierKeeper.SetSupplier(ctx, supplier)

				logger.Info(
					"Successfully migrated supplier data",
					"supplier_address", supplier.OperatorAddress,
				)
			}

			logger.Info("Starting module migrations section", "upgrade_plan_name", Upgrade_0_0_14_PlanName)
			vm, err := mm.RunMigrations(ctx, configurator, vm)
			if err != nil {
				logger.Error("Failed to run migrations",
					"upgrade_plan_name", Upgrade_0_0_14_PlanName,
					"error", err)
				return vm, err
			}

			logger.Info("Successfully completed upgrade handler", "upgrade_plan_name", Upgrade_0_0_14_PlanName)
			return vm, nil
		}
	},
	// No migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},
}
