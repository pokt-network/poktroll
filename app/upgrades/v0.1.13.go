package upgrades

import (
	"context"

	cosmoslog "cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/pokt-network/poktroll/app/keepers"
)

const (
	Upgrade_0_1_13_PlanName = "v0.1.13"
)

// Upgrade_0_1_13 handles the upgrade to release `v0.1.13`.
// This upgrade adds:
// - the `morse_account_claiming_enabled` migration module param
//   - Set to `true` by default
//
// https://github.com/pokt-network/poktroll/compare/v0.1.12..v0.1.13
var Upgrade_0_1_13 = Upgrade{
	PlanName: Upgrade_0_1_13_PlanName,
	// No migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},

	// Upgrade Handler
	CreateUpgradeHandler: func(
		mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		// Add new parameters by:
		// 1. Inspecting the diff between v0.1.12...v0.1.13
		// 2. Manually inspect changes in ignite's config.yml
		// 3. Update the upgrade handler here accordingly
		// Ref: https://github.com/pokt-network/poktroll/compare/v0.1.12...v0.1.13
		applyNewParameters := func(ctx context.Context, logger cosmoslog.Logger) (err error) {
			logger.Info("Starting parameter updates", "upgrade_plan_name", Upgrade_0_1_13_PlanName)

			// Get the current migration module params
			migrationParams := keepers.MigrationKeeper.GetParams(ctx)

			// Set morse_account_claiming_enabled to true by default.
			migrationParams.MorseAccountClaimingEnabled = true

			// Ensure that the new parameters are valid
			if err = migrationParams.Validate(); err != nil {
				logger.Error("Failed to validate migration params", "error", err)
				return err
			}

			// ALL parameters in the migration module must be specified when
			// setting parameters, even if just one is being CRUDed.
			err = keepers.MigrationKeeper.SetParams(ctx, migrationParams)
			if err != nil {
				logger.Error("Failed to set migration params", "error", err)
				return err
			}
			logger.Info("Successfully updated migration params", "new_params", migrationParams)

			return nil
		}

		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			logger := cosmostypes.UnwrapSDKContext(ctx).Logger()

			if err := applyNewParameters(ctx, logger); err != nil {
				return vm, err
			}

			return vm, nil
		}
	},
}
