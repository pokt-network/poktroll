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
// This is planned to be issued on both Pocket Network's Shannon Alpha & Beta TestNets as well as Mainnet.
var Upgrade_0_1_2 = Upgrade{
	PlanName: Upgrade_0_1_2_PlanName,
	// No state changes in this upgrade.
	CreateUpgradeHandler: func(mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		// Adds new parameters using ignite's config.yml as a reference.
		// https://github.com/pokt-network/poktroll/compare/v0.1.1...f1d354d
		applyNewParameters := func(ctx context.Context) (err error) {
			logger := cosmosTypes.UnwrapSDKContext(ctx).Logger()
			logger.Info("Starting parameter updates", "upgrade_plan_name", Upgrade_0_1_2_PlanName)

			// Set waive_morse_claim_gas_fees to true
			migrationParams := migrationtypes.Params{
				WaiveMorseClaimGasFees: true,
			}

			// Ensure that the new parameters are valid
			if err = migrationParams.Validate(); err != nil {
				logger.Error("Failed to validate migration params", "error", err)
				return err
			}

			// ALL parameters must be present when setting params.
			err = keepers.MigrationKeeper.SetParams(ctx, migrationParams)
			if err != nil {
				logger.Error("Failed to set migration params", "error", err)
				return err
			}
			logger.Info("Successfully updated migration params", "new_params", migrationParams)

			return nil
		}

		// Adds new authz authorizations from the diff:
		// https://github.com/pokt-network/poktroll/compare/v0.1.1...f1d354d
		applyNewAuthorizations := func(ctx context.Context) (err error) {
			// Validate before/after with:
			// `pocketd q authz grants-by-granter pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t --node=<netsork_rpc_url>`
			grantAuthorizationMessages := []string{
				"/pocket.migration.MsgUpdateParam",
				"/pocket.migration.MsgImportMorseClaimableAccounts",
			}

			expiration, err := time.Parse(time.RFC3339, "2500-01-01T00:00:00Z")
			if err != nil {
				return fmt.Errorf("failed to parse time: %w", err)
			}

			response, err := keepers.UpgradeKeeper.Authority(ctx, &upgradetypes.QueryAuthorityRequest{})
			if err != nil {
				return fmt.Errorf("failed to query authority address: %w", err)
			}
			authorityAddress := response.GetAddress()

			// Get the grantee address for the current network.
			pnfAddress := NetworkAuthzGranteeAddress[cosmosTypes.UnwrapSDKContext(ctx).ChainID()]

			for _, msg := range grantAuthorizationMessages {
				err = keepers.AuthzKeeper.SaveGrant(
					ctx,
					cosmosTypes.AccAddress(pnfAddress),
					cosmosTypes.AccAddress(authorityAddress),
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
			logger.Info("Starting upgrade handler", "upgrade_plan_name", Upgrade_0_1_2_PlanName)

			err := applyNewParameters(ctx)
			if err != nil {
				logger.Error("Failed to apply new parameters",
					"upgrade_plan_name", Upgrade_0_1_2_PlanName,
					"error", err)
				return vm, err
			}

			err = applyNewAuthorizations(ctx)
			if err != nil {
				logger.Error("Failed to apply new authorizations",
					"upgrade_plan_name", Upgrade_0_1_2_PlanName,
					"error", err)
				return vm, err
			}

			logger.Info("Starting module migrations section", "upgrade_plan_name", Upgrade_0_1_2_PlanName)
			vm, err = mm.RunMigrations(ctx, configurator, vm)
			if err != nil {
				logger.Error("Failed to run migrations",
					"upgrade_plan_name", Upgrade_0_1_2_PlanName,
					"error", err)
				return vm, err
			}

			logger.Info("Successfully completed upgrade", "upgrade_plan_name", Upgrade_0_1_2_PlanName)
			return vm, nil
		}
	},
	// No migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},
}
