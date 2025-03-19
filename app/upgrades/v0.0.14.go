package upgrades

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cosmosTypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/pokt-network/poktroll/app/keepers"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
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
			suppliers := supplierKeeper.GetAllSuppliers(ctx)

			logger.Info("All suppliers", "suppliers", len(suppliers))

			for _, supplier := range suppliers {
				// Only migrate if the map has data to migrate
				if len(supplier.ServicesActivationHeightsMap) > 0 {
					logger.Info(
						"Migrating services activation heights",
						"supplier_address", supplier.OperatorAddress,
						"services_count", len(supplier.ServicesActivationHeightsMap),
					)

					// For each height in the activation heights map, create a service config update
					heightsMap := make(map[uint64]bool)
					for _, height := range supplier.ServicesActivationHeightsMap {
						heightsMap[height] = true
					}

					// Convert to service config updates
					for height := range heightsMap {
						// Check if we already have an entry for this height
						exists := false
						for _, existing := range supplier.ServiceConfigHistory {
							if existing.EffectiveBlockHeight == height {
								exists = true
								break
							}
						}

						// Only add if it doesn't exist
						if !exists {
							configUpdate := &sharedtypes.ServiceConfigUpdate{
								Services:             supplier.Services,
								EffectiveBlockHeight: height,
							}
							supplier.ServiceConfigHistory = append(supplier.ServiceConfigHistory, configUpdate)
						}
					}

					// Clear the activation heights map after migration
					supplier.ServicesActivationHeightsMap = make(map[string]uint64)

					// Update the supplier with the migrated data
					supplierKeeper.SetSupplier(ctx, supplier)

					logger.Info(
						"Successfully migrated supplier data",
						"supplier_address", supplier.OperatorAddress,
					)
				}
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
