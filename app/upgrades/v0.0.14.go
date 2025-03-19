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
				logger.Info("Migrating supplier data", "supplier_address", supplier.OperatorAddress)

				// Only migrate if the map has data to migrate
				if len(supplier.ServicesActivationHeightsMap) > 0 {
					logger.Info(
						"Migrating services activation heights",
						"supplier_address", supplier.OperatorAddress,
						"services_count", len(supplier.ServicesActivationHeightsMap),
					)

					// If the ServiceConfigHistory is empty, we need to initialize it
					if len(supplier.ServiceConfigHistory) == 0 {
						// Create an initial ServiceConfigUpdate with all current services
						// and the earliest activation height
						var earliestHeight uint64 = ^uint64(0) // Max uint64 value
						for _, height := range supplier.ServicesActivationHeightsMap {
							if height < earliestHeight {
								earliestHeight = height
							}
						}

						// Create a config update with all current services
						initialConfigUpdate := &sharedtypes.ServiceConfigUpdate{
							Services:             supplier.Services,
							EffectiveBlockHeight: earliestHeight,
						}

						supplier.ServiceConfigHistory = append(supplier.ServiceConfigHistory, initialConfigUpdate)
					}

					// Now process each service in the activation heights map
					// We'll convert each service activation to a service config update
					for serviceID, activationHeight := range supplier.ServicesActivationHeightsMap {
						// Find the service in current services list
						var targetService *sharedtypes.SupplierServiceConfig
						for _, svc := range supplier.Services {
							if svc.ServiceId == serviceID {
								targetService = svc
								break
							}
						}

						// Skip if we don't have this service in our current services list
						if targetService == nil {
							logger.Info(
								"Service not found in current services, skipping",
								"supplier_address", supplier.OperatorAddress,
								"service_id", serviceID,
							)
							continue
						}

						// Create a ServiceConfigUpdate for this service activation
						configUpdate := &sharedtypes.ServiceConfigUpdate{
							// Include all current services in the update
							Services:             supplier.Services,
							EffectiveBlockHeight: activationHeight,
						}

						// Add to history if it doesn't already exist with that activation height
						exists := false
						for _, existing := range supplier.ServiceConfigHistory {
							if existing.EffectiveBlockHeight == activationHeight {
								exists = true
								break
							}
						}

						if !exists {
							supplier.ServiceConfigHistory = append(supplier.ServiceConfigHistory, configUpdate)
						}
					}

					// Clear the activation heights map after migration
					supplier.ServicesActivationHeightsMap = make(map[string]uint64)

					// Update the supplier with the migrated data
					supplierKeeper.SetSupplier(ctx, supplier)

					logger.Info(
						"Successfully migrated supplier services activation data",
						"supplier_address", supplier.OperatorAddress,
						"service_config_history_count", len(supplier.ServiceConfigHistory),
					)
				} else {
					logger.Info(
						"No services activation heights to migrate",
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
