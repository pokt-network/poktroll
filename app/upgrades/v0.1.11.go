package upgrades

import (
	"context"

	cosmoslog "cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/pokt-network/poktroll/app/keepers"
	"github.com/pokt-network/poktroll/app/volatile"
)

const (
	Upgrade_0_1_11_PlanName = "v0.1.11"
)

// Upgrade_0_1_11 handles the upgrade to release `v0.1.11`.
// This upgrade adds:
// - the `allow_morse_account_import_overwrite` migration module param
//   - Set to `false` by default
//   - Set to `true` on Alpha and Beta TestNets
//
// - a corresponding authz grant
// - new `MsgRecoverMorseAccount` message with empty message handlers (scaffolding)
// https://github.com/pokt-network/poktroll/compare/v0.1.10..v0.1.11
var Upgrade_0_1_11 = Upgrade{
	PlanName: Upgrade_0_1_11_PlanName,
	// No migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},

	// Upgrade Handler
	CreateUpgradeHandler: func(
		mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		// Add new parameters by:
		// 1. Inspecting the diff between v0.1.10...v0.1.11
		// 2. Manually inspect changes in ignite's config.yml
		// 3. Update the upgrade handler here accordingly
		// Ref: https://github.com/pokt-network/poktroll/compare/v0.1.10...v0.1.11
		applyNewParameters := func(ctx context.Context, logger cosmoslog.Logger) (err error) {
			logger.Info("Starting parameter updates", "upgrade_plan_name", Upgrade_0_1_11_PlanName)

			// Get the current migration module params
			migrationParams := keepers.MigrationKeeper.GetParams(ctx)

			// Set allow_morse_account_import_overwrite to:
			// - True for Alpha and Beta TestNets
			// - False for ALL other chain IDs (e.g. MainNet)
			switch cosmostypes.UnwrapSDKContext(ctx).ChainID() {
			case volatile.AlphaTestNetChainId, volatile.BetaTestNetChainId:
				migrationParams.AllowMorseAccountImportOverwrite = true
			default:
				migrationParams.AllowMorseAccountImportOverwrite = false
			}

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

			// Adds new authz authorizations from the diff:
			// https://github.com/pokt-network/poktroll/compare/v0.1.10...v0.1.11
			grantAuthorizationMessages := []string{"/pocket.migration.MsgRecoverMorseAccount"}
			if err := applyNewAuthorizations(ctx, keepers, logger, grantAuthorizationMessages); err != nil {
				return vm, err
			}

			if err := applyNewParameters(ctx, logger); err != nil {
				return vm, err
			}

			return vm, nil
		}
	},
}
