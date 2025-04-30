package upgrades

import (
	"context"
	"slices"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/pokt-network/poktroll/app/keepers"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const (
	Upgrade_0_1_8_PlanName = "v0.1.8"
)

// Upgrade_0_1_8 handles the upgrade to release `v0.1.8`.
// This is planned to be issued on both Pocket Network's Shannon Alpha, Beta TestNets
// It is an upgrade intended to enable suppliers service config indexing and more granular hydration.
// TODO_FOLLOWUP(#1230, @red-0ne): Update the github link from main to v0.1.8 once the upgrade is released.
// https://github.com/pokt-network/poktroll/compare/v0.1.7..v0.1.8
var Upgrade_0_1_8 = Upgrade{
	PlanName: Upgrade_0_1_8_PlanName,
	// No migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},

	// Upgrade Handler
	CreateUpgradeHandler: func(
		mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			logger := cosmostypes.UnwrapSDKContext(ctx).Logger().With("upgrade_plan_name", Upgrade_0_1_8_PlanName)
			logger.Info("Starting upgrade handler")

			logger.Info("Indexing suppliers service configs")
			if err := indexSuppliersServiceConfigs(ctx, keepers, logger); err != nil {
				return vm, err
			}

			return vm, nil
		}
	},
}

// indexSuppliersServiceConfigs indexes the service config updates for all suppliers.
// It processes the deprecated service config history of each supplier, converting
// it into the new format.
func indexSuppliersServiceConfigs(ctx context.Context, keepers *keepers.Keepers, logger log.Logger) error {
	// Get all deprecated suppliers from the store.
	deprecatedSuppliers := keepers.SupplierKeeper.GetAllDeprecatedSuppliers(ctx)
	for _, deprecatedSupplier := range deprecatedSuppliers {
		logger.Info("Indexing supplier", "operator_address", deprecatedSupplier.OperatorAddress)

		// serviceConfigUpdates is a slice of service config updates that will be
		// assigned to the supplier after the migration.
		serviceConfigUpdates := make([]*sharedtypes.ServiceConfigUpdate, 0)

		// latestServiceConfigUpdateIndex is a map that keeps track of the latest
		// service config update index for each service.
		// This is used to mark the previous service config update as deactivated
		// when a new one is found.
		latestServiceConfigUpdateIndex := make(map[string]int)

		serviceConfigHistory := deprecatedSupplier.ServiceConfigHistory
		slices.SortFunc(serviceConfigHistory, func(i, j *sharedtypes.ServiceConfigUpdateDeprecated) int {
			return int(i.EffectiveBlockHeight - j.EffectiveBlockHeight)
		})

		for i, deprecatedServiceConfigUpdates := range serviceConfigHistory {
			// In the deprecated service config history, the effective block height is
			// global for all services in the update.
			// In the new service config history, the effective block height is per service
			// which allows for more granular updates.
			effectiveBlockHeight := int64(deprecatedServiceConfigUpdates.EffectiveBlockHeight)

			for _, service := range deprecatedServiceConfigUpdates.Services {
				// Ensure that the most recently active service config update is the only one
				// active for each service.
				if idx, ok := latestServiceConfigUpdateIndex[service.ServiceId]; ok {
					if effectiveBlockHeight >= serviceConfigUpdates[idx].ActivationHeight {
						// The previous service config update is now superceded by the new one.
						// Mark it as deactivated.
						serviceConfigUpdates[idx].DeactivationHeight = effectiveBlockHeight
					}
				}

				// Create a new service config update object with the effective block height
				serviceConfigUpdate := &sharedtypes.ServiceConfigUpdate{
					OperatorAddress:  deprecatedSupplier.OperatorAddress,
					Service:          service,
					ActivationHeight: effectiveBlockHeight,
				}
				serviceConfigUpdates = append(serviceConfigUpdates, serviceConfigUpdate)

				// Update the latest service config update index for this service
				// so it can be marked as deactivated if a newer one is found.
				latestServiceConfigUpdateIndex[service.ServiceId] = i
			}
		}

		// Create a new supplier object with the updated service config history
		supplier := sharedtypes.Supplier{
			ServiceConfigHistory: serviceConfigUpdates,

			// The rest of the supplier fields remain unchanged.
			// They are copied from the deprecated supplier object.
			OperatorAddress:         deprecatedSupplier.OperatorAddress,
			OwnerAddress:            deprecatedSupplier.OwnerAddress,
			Stake:                   deprecatedSupplier.Stake,
			Services:                deprecatedSupplier.Services,
			UnstakeSessionEndHeight: deprecatedSupplier.UnstakeSessionEndHeight,
		}

		// SetSupplier will automatically index the supplier service configs using
		// the new ServiceConfigHistory.
		keepers.SupplierKeeper.SetSupplier(ctx, supplier)
	}
	return nil
}
