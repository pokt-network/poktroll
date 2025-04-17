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
)

const Upgrade_0_1_4_PlanName = "v0.1.4"

// Upgrade_0_1_4 handles the upgrade to release `v0.1.4`.
// This is planned to be issued on both Pocket Network's Shannon Alpha & Beta TestNets as well as Mainnet.
var Upgrade_0_1_4 = Upgrade{
	// Plan Name
	PlanName: Upgrade_0_1_4_PlanName,

	// No migrations or state changes in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},

	// Upgrade Handler
	CreateUpgradeHandler: func(mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		// Adds new authz authorizations from the diff:
		// https://github.com/pokt-network/poktroll/compare/v0.1.1...f1d354d
		applyNewAuthorizations := func(ctx context.Context) (err error) {
			// Validate before/after with:
			// 	pocketd q authz grants-by-granter pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t --node=<network_rpc_url>
			// Find a network RPC URL here: https://dev.poktroll.com/category/explorers-faucets-wallets-and-more
			grantAuthorizationMessages := []string{
				"/pocket.migration.MsgUpdateParam",
				"/pocket.migration.MsgImportMorseClaimableAccounts",
			}

			expiration, err := time.Parse(time.RFC3339, "2500-01-01T00:00:00Z")
			if err != nil {
				return fmt.Errorf("failed to parse time: %w", err)
			}

			// Get the granter address of the migration module (i.e. authority)
			granterAddr := keepers.MigrationKeeper.GetAuthority()

			// Get the grantee address for the current network (i.e. pnf or grove)
			granteeAddr := NetworkAuthzGranteeAddress[cosmosTypes.UnwrapSDKContext(ctx).ChainID()]

			for _, msg := range grantAuthorizationMessages {
				err = keepers.AuthzKeeper.SaveGrant(
					ctx,
					cosmosTypes.AccAddress(granteeAddr),
					cosmosTypes.AccAddress(granterAddr),
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
			logger.Info("Starting upgrade handler", "upgrade_plan_name", Upgrade_0_1_4_PlanName)

			// Apply authorization changes
			err := applyNewAuthorizations(ctx)
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
}
