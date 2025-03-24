package upgrades

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cosmosTypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/pokt-network/pocket/app/keepers"
	migrationtypes "github.com/pokt-network/pocket/x/migration/types"
	sharedtypes "github.com/pokt-network/pocket/x/shared/types"
)

const Upgrade_0_0_13_PlanName = "v0.0.13"

// Upgrade_0_0_13 handles the upgrade to release `v0.0.13`.
// This is planned to be issued on both Pocket Network's Shannon Alpha & Beta TestNets.
var Upgrade_0_0_13 = Upgrade{
	PlanName: Upgrade_0_0_13_PlanName,
	CreateUpgradeHandler: func(mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		applyNewParameters := func(ctx context.Context) (err error) {
			logger := cosmosTypes.UnwrapSDKContext(ctx).Logger()
			logger.Info("Starting parameter updates", "upgrade_plan_name", Upgrade_0_0_13_PlanName)

			sharedParams := keepers.SharedKeeper.GetParams(ctx)
			sharedParams.GatewayUnbondingPeriodSessions = sharedtypes.DefaultGatewayUnbondingPeriodSessions

			// Ensure that the new parameters are valid
			if err = sharedParams.ValidateBasic(); err != nil {
				logger.Error("Failed to validate shared params", "error", err)
				return err
			}

			err = keepers.SharedKeeper.SetParams(ctx, sharedParams)
			if err != nil {
				logger.Error("Failed to set shared params", "error", err)
				return err
			}
			logger.Info("Successfully updated shared params", "new_params", sharedParams)

			return nil
		}

		// Returns the upgrade handler for v0.0.13
		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			logger := cosmosTypes.UnwrapSDKContext(ctx).Logger()
			logger.Info("Starting upgrade handler", "upgrade_plan_name", Upgrade_0_0_13_PlanName)

			logger.Info("Starting parameter updates section", "upgrade_plan_name", Upgrade_0_0_13_PlanName)
			// Update all governance parameter changes.
			// This includes adding params, removing params and changing values of existing params.
			err := applyNewParameters(ctx)
			if err != nil {
				logger.Error("Failed to apply new parameters",
					"upgrade_plan_name", Upgrade_0_0_13_PlanName,
					"error", err)
				return vm, err
			}

			logger.Info("Starting module migrations section", "upgrade_plan_name", Upgrade_0_0_13_PlanName)
			vm, err = mm.RunMigrations(ctx, configurator, vm)
			if err != nil {
				logger.Error("Failed to run migrations",
					"upgrade_plan_name", Upgrade_0_0_13_PlanName,
					"error", err)
				return vm, err
			}

			logger.Info("Successfully completed upgrade handler", "upgrade_plan_name", Upgrade_0_0_13_PlanName)
			return vm, nil
		}
	},
	// Add the migration module KVStore in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{
		// Add the new morse migration module KVStore to the upgrade plan.
		Added: []string{migrationtypes.StoreKey},
	},
}
