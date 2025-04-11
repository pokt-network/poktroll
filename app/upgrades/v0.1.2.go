package upgrades

import (
	"context"
	"fmt"
	"time"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cosmosTypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/authz"

	"github.com/pokt-network/poktroll/app/keepers"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

const Upgrade_0_1_2_PlanName = "v0.1.2"

// Upgrade_0_1_2 handles the upgrade to release `v0.1.2`.
// This is planned to be issued on both Pocket Network's Shannon Alpha & Beta TestNets.
var Upgrade_0_1_2 = Upgrade{
	PlanName: Upgrade_0_1_2_PlanName,
	// No state changes in this upgrade.
	CreateUpgradeHandler: func(mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		// Adds new parameters using ignite's config.yml as a reference. Assuming we don't need any other parameters.
		// https://github.com/pokt-network/poktroll/compare/v0.1.1...v0.1.2-rc
		applyNewParameters := func(ctx context.Context) (err error) {
			logger := cosmosTypes.UnwrapSDKContext(ctx).Logger()
			logger.Info("Starting parameter updates for v0.1.2")

			// Set waive_morse_claim_gas_fees to true
			migrationParams := migrationtypes.Params{
				WaiveMorseClaimGasFees: true,
			}

			// ALL parameters must be present when setting params.
			err = keepers.MigrationKeeper.SetParams(ctx, migrationParams)
			if err != nil {
				logger.Error("Failed to set migration params", "error", err)
				return err
			}
			logger.Info("Successfully updated migration params", "new_params", migrationParams)

			return
		}

		// TODO_IN_THIS_COMMIT: update comment once tags/hashes are available/known.
		// Adds new authz authorizations from the diff:
		// https://github.com/pokt-network/poktroll/compare/v0.1.1...ff76430
		applyNewAuthorizations := func(ctx context.Context) (err error) {
			// Validate before/after with:
			// `pocketd q authz grants-by-granter pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t --node=https://testnet-validated-validator-rpc.poktroll.com/`
			grantAuthorizationMessages := []string{
				"/pocket.migration.MsgUpdateParam",
			}

			expiration, err := time.Parse(time.RFC3339, "2500-01-01T00:00:00Z")
			if err != nil {
				return fmt.Errorf("failed to parse time: %w", err)
			}

			for _, msg := range grantAuthorizationMessages {
				err = keepers.AuthzKeeper.SaveGrant(
					ctx,
					cosmosTypes.AccAddress(AlphaTestNetPnfAddress),
					cosmosTypes.AccAddress(AlphaTestNetAuthorityAddress),
					authz.NewGenericAuthorization(msg),
					&expiration,
				)
				if err != nil {
					return fmt.Errorf("failed to save grant for message %s: %w", msg, err)
				}
			}
			return
		}

		// Returns the upgrade handler for v0.1.2
		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			logger := cosmosTypes.UnwrapSDKContext(ctx).Logger()
			logger.Info("Starting v0.1.2 upgrade handler")

			err := applyNewParameters(ctx)
			if err != nil {
				logger.Error("Failed to apply new parameters", "error", err)
				return vm, err
			}

			err = applyNewAuthorizations(ctx)
			if err != nil {
				return vm, err
			}

			logger.Info("Running module migrations")
			vm, err = mm.RunMigrations(ctx, configurator, vm)
			if err != nil {
				logger.Error("Failed to run migrations", "error", err)
				return vm, err
			}

			logger.Info("Successfully completed v0.1.2 upgrade handler")
			return vm, nil
		}
	},
	// No migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},
}
