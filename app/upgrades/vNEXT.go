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

// TODO_NEXT_UPGRADE: Rename NEXT with the appropriate next
// upgrade version number and update comment versions.

const (
	Upgrade_NEXT_PlanName = "vNEXT"
)

// Upgrade_NEXT handles the upgrade to release `vNEXT`.
// This upgrade adds:
// - the `compute_unit_cost_granularity` shared module param
//
// https://github.com/pokt-network/poktroll/compare/vPREV..vNEXT
var Upgrade_NEXT = Upgrade{
	PlanName: Upgrade_NEXT_PlanName,
	// No KVStore migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},

	// Upgrade Handler
	CreateUpgradeHandler: func(
		mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		// Add new parameters by:
		// 1. Inspecting the diff between vPREV...vNEXT
		// 2. Manually inspect changes in ignite's config.yml
		// 3. Update the upgrade handler here accordingly
		// Ref: https://github.com/pokt-network/poktroll/compare/vPREV...vNEXT

		applyComputeUnitsCostGranularity := func(ctx context.Context, logger cosmoslog.Logger) (err error) {
			logger.Info("Starting shared module parameter updates", "upgrade_plan_name", Upgrade_NEXT_PlanName)

			// Get the current shared module params
			sharedParams := keepers.SharedKeeper.GetParams(ctx)

			// Set compute_unit_cost_granularity to 1e6 making compute_units_to_tokens_multiplier
			// to be denominated in pPOKT (i.e. 1/1e6 uPOKT)
			sharedParams.ComputeUnitCostGranularity = 1e6
			// Maintain the compute_units_to_tokens_multiplier uPOKT value,
			// Update it to be denominated in 1/compute_unit_cost_granularity uPOKT
			// by multiplying it by the compute_unit_cost_granularity
			sharedParams.ComputeUnitsToTokensMultiplier *= sharedParams.ComputeUnitCostGranularity

			// Ensure that the new parameters are valid
			if err = sharedParams.ValidateBasic(); err != nil {
				logger.Error("Failed to validate shared module params", "error", err)
				return err
			}

			// ALL parameters in the shared module must be specified when
			// setting parameters, even if just one is being CRUDed.
			err = keepers.SharedKeeper.SetParams(ctx, sharedParams)
			if err != nil {
				logger.Error("Failed to set shared module params", "error", err)
				return err
			}
			logger.Info("Successfully updated shared module params", "new_params", sharedParams)

			return nil
		}

		applyMorseAccountClaimingEnabled := func(ctx context.Context, logger cosmoslog.Logger) (err error) {
			logger.Info("Starting migration module parameter updates", "upgrade_plan_name", Upgrade_NEXT_PlanName)

			// Get the current migration module params
			migrationParams := keepers.MigrationKeeper.GetParams(ctx)

			// Set morse_account_claiming_enabled to true by default.
			migrationParams.MorseAccountClaimingEnabled = true

			// Ensure that the new parameters are valid
			if err = migrationParams.Validate(); err != nil {
				logger.Error("Failed to validate migration module params", "error", err)
				return err
			}

			// ALL parameters in the migration module must be specified when
			// setting parameters, even if just one is being CRUDed.
			err = keepers.MigrationKeeper.SetParams(ctx, migrationParams)
			if err != nil {
				logger.Error("Failed to set migration module params", "error", err)
				return err
			}
			logger.Info("Successfully updated migration module params", "new_params", migrationParams)

			return nil
		}

		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			logger := cosmostypes.UnwrapSDKContext(ctx).Logger()

			if err := applyComputeUnitsCostGranularity(ctx, logger); err != nil {
				return vm, err
			}

			if err := applyMorseAccountClaimingEnabled(ctx, logger); err != nil {
				return vm, err
			}

			return vm, nil
		}
	},
}
