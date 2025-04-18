package upgrades

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cosmosTypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/pokt-network/poktroll/app/keepers"
)

const Upgrade_0_1_4_PlanName = "v0.1.4"

// Upgrade_0_1_4 handles the upgrade to release `v0.1.4`.
// A follow up to `v0.1.2` that has to be re-applied due to the issues outlined here:  https://github.com/cosmos/cosmos-sdk/pull/24548
var Upgrade_0_1_4 = Upgrade{
	// Plan Name
	PlanName: Upgrade_0_1_4_PlanName,

	// No migrations or state changes in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},

	// Upgrade Handler
	CreateUpgradeHandler: func(
		mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {

		// Returns the upgrade handler for v0.1.4
		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			logger := cosmosTypes.UnwrapSDKContext(ctx).Logger().With("upgrade_plan_name", Upgrade_0_1_4_PlanName)
			logger.Info("Starting upgrade handler")

			// *** Apply authorization changes ***

			// Same list as in v0.1.2 due to the issues outlined here: https://github.com/cosmos/cosmos-sdk/pull/24548
			grantAuthorizationMessages := []string{
				"/pocket.migration.MsgUpdateParams",
				"/pocket.migration.MsgImportMorseClaimableAccounts",
			}

			logger.Info("Starting authorization updates")
			err := applyNewAuthorizations(ctx, keepers, logger, grantAuthorizationMessages)
			if err != nil {
				logger.Error("Failed to apply new authorizations", "error", err)
				return vm, err
			}
			logger.Info("Successfully completed authorization updates")

			// Run module migrations
			logger.Info("Starting module migrations section")
			vm, err = mm.RunMigrations(ctx, configurator, vm)
			if err != nil {
				logger.Error("Failed to run migrations", "error", err)
				return vm, err
			}
			logger.Info("Successfully completed module migrations")

			logger.Info("Successfully completed upgrade")
			return vm, nil
		}
	},
}
