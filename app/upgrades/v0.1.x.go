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

		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			if err := applyNewAuthorizations(ctx); err != nil {
				return vm, err
			}

			return vm, nil
		}
	},
}
