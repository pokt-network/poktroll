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
	Upgrade_0_1_33_PlanName = "v0.1.33"
)

// Upgrade_0_1_33 handles the upgrade to release `v0.1.33`.
// This upgrade fixes:
// - removeApplicationUndelegationIndex deleting from the wrong store (delegation
//   store instead of undelegation store), causing orphaned undelegation index entries
//   when applications with pending undelegations were removed (unstaked/transferred).
//   Bug introduced in PR #1263 (v0.1.31). The upgrade handler cleans up any
//   orphaned entries accumulated since v0.1.31.
var Upgrade_0_1_33 = Upgrade{
	PlanName: Upgrade_0_1_33_PlanName,
	// No KVStore migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},

	// Upgrade Handler
	CreateUpgradeHandler: func(
		mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {

		// Clean up orphaned undelegation index entries.
		// These are index entries that reference applications which no longer exist,
		// caused by removeApplicationUndelegationIndex deleting from getDelegationStore()
		// instead of getUndelegationStore(). Introduced in PR #1263 (v0.1.31).
		cleanupOrphanedUndelegationIndexes := func(ctx context.Context, logger cosmoslog.Logger) error {
			logger.Info("Cleaning up orphaned application undelegation indexes")

			count, err := keepers.ApplicationKeeper.CleanupOrphanedUndelegationIndexes(ctx)
			if err != nil {
				logger.Error("Failed to cleanup orphaned undelegation indexes", "error", err)
				return err
			}

			logger.Info("Cleaned up orphaned application undelegation indexes",
				"orphaned_entries_removed", count,
			)
			return nil
		}

		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
			logger := sdkCtx.Logger()

			if err := cleanupOrphanedUndelegationIndexes(ctx, logger); err != nil {
				return vm, err
			}

			return vm, nil
		}
	},
}
