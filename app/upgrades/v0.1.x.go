package upgrades

import (
	"context"
	"fmt"
	"time"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/authz"

	"github.com/pokt-network/poktroll/app/keepers"
)

// TODO_MAINNET_MIGRATION(@bryanchriswhite): Update the upgrade version numbers, diffs, hashes, etc. and rename this file.

const (
	Upgrade_0_1_x_PlanName = "v0.1.x"
)

// Upgrade_0_1_x handles the upgrade to release `v0.1.x`.
// This is planned to be issued on both Pocket Network's Shannon Alpha, Beta TestNets as well as MainNet.
// It is an upgrade intended to reduce the memory footprint when iterating over Suppliers and Applications.
// https://github.com/pokt-network/poktroll/compare/v0.1.x..abc123
var Upgrade_0_1_x = Upgrade{
	PlanName: Upgrade_0_1_x_PlanName,
	// No migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},

	// Upgrade Handler
	CreateUpgradeHandler: func(
		mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		// Adds new authz authorizations from the diff:
		// https://github.com/pokt-network/poktroll/compare/v0.1.x...abc123
		applyNewAuthorizations := func(ctx context.Context) (err error) {
			// Validate before/after with:
			// 	pocketd q authz grants-by-granter pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t --node=<network_rpc_url>
			// Find a network RPC URL here: https://dev.poktroll.com/category/explorers-faucets-wallets-and-more
			grantAuthorizationMessages := []string{
				"/pocket.migration.MsgRecoverMorseAccount",
			}

			expiration, err := time.Parse(time.RFC3339, "2500-01-01T00:00:00Z")
			if err != nil {
				return fmt.Errorf("failed to parse time: %w", err)
			}

			// Get the granter address of the migration module (i.e. authority)
			granterAddr := keepers.MigrationKeeper.GetAuthority()

			// Get the grantee address for the current network (i.e. pnf or grove)
			granteeAddr := NetworkAuthzGranteeAddress[cosmostypes.UnwrapSDKContext(ctx).ChainID()]

			for _, msg := range grantAuthorizationMessages {
				err = keepers.AuthzKeeper.SaveGrant(
					ctx,
					cosmostypes.AccAddress(granteeAddr),
					cosmostypes.AccAddress(granterAddr),
					authz.NewGenericAuthorization(msg),
					&expiration,
				)
				if err != nil {
					return fmt.Errorf("failed to save grant for message %s: %w", msg, err)
				}
			}
			return
		}

		// Add new parameters by:
		// 1. Inspecting the diff between v0.1.x...abc123
		// 2. Manually inspect changes in ignite's config.yml
		// 3. Update the upgrade handler here accordingly
		// Ref: https://github.com/pokt-network/poktroll/compare/v0.1.x...abc123
		applyNewParameters := func(ctx context.Context) (err error) {
			logger := cosmostypes.UnwrapSDKContext(ctx).Logger()
			logger.Info("Starting parameter updates", "upgrade_plan_name", Upgrade_0_1_x_PlanName)

			// Get the current migration module params
			migrationParams := keepers.MigrationKeeper.GetParams(ctx)

			// Set allow_morse_account_import_overwrite to:
			// - True for Alpha and Beta TestNets
			// - False for ALL other chain IDs (e.g. MainNet)
			switch cosmostypes.UnwrapSDKContext(ctx).ChainID() {
			case AlphaTestNetChainId, BetaTestNetChainId:
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
			if err := applyNewAuthorizations(ctx); err != nil {
				return vm, err
			}

			if err := applyNewParameters(ctx); err != nil {
				return vm, err
			}

			return vm, nil
		}
	},
}
