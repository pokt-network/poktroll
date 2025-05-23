package upgrades

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/pokt-network/poktroll/app/keepers"
)

const Upgrade_0_1_9_PlanName = "v0.1.9"

// Upgrade_0_1_9 handles the upgrade to release `v0.1.9`.
// This is planned to be issued on both Pocket Network's Shannon Alpha, Beta TestNets as well as MainNet.
// It is an upgrade intended to reduce the claim settlement processing time by
// caching the redundant data in the store and avoiding unnecessary marshaling,
// indexing and data retrieval.
// https://github.com/pokt-network/poktroll/compare/v0.1.8..v0.1.9
var Upgrade_0_1_9 = Upgrade{
	PlanName: Upgrade_0_1_9_PlanName,
	// No migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},

	// Upgrade Handler
	CreateUpgradeHandler: func(
		mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			logger := cosmostypes.UnwrapSDKContext(ctx).Logger().With("upgrade_plan_name", Upgrade_0_1_9_PlanName)
			logger.Info("Starting upgrade handler")

			logger.Info("re-indexing suppliers service configs")
			keepers.SupplierKeeper.MigrateSupplierServiceConfigIndexes(ctx)

			return vm, nil
		}
	},
}
