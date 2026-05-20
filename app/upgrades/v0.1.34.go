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
	Upgrade_0_1_34_PlanName = "v0.1.34"
)

// Upgrade_0_1_34 handles the upgrade to release `v0.1.34`.
// This upgrade adds:
//   - Deduplicate supplier rev share addresses in service config history.
//
// NOTE: Application service config history (added in this release for
// deterministic historical session queries) requires NO migration: an empty
// history means the application never changed its service config, and
// GetActiveServiceConfigs falls back to the flat ServiceConfigs snapshot for
// such apps. History is written lazily, only when an app actually swaps service.
var Upgrade_0_1_34 = Upgrade{
	PlanName: Upgrade_0_1_34_PlanName,
	// No KVStore migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},

	// Upgrade Handler
	CreateUpgradeHandler: func(
		mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {

		deduplicateSupplierRevShareAddresses := func(ctx context.Context, logger cosmoslog.Logger) error {
			logger.Info("Deduplicating supplier rev share addresses")

			count, err := keepers.SupplierKeeper.DeduplicateSupplierRevShareAddresses(ctx)
			if err != nil {
				logger.Error("Failed to deduplicate supplier rev share addresses", "error", err)
				return err
			}

			logger.Info("Deduplicated supplier rev share addresses",
				"modified_suppliers", count,
			)
			return nil
		}

		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
			logger := sdkCtx.Logger()

			if err := deduplicateSupplierRevShareAddresses(ctx, logger); err != nil {
				return vm, err
			}

			return vm, nil
		}
	},
}
