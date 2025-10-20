package upgrades

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/pokt-network/poktroll/app/keepers"
)

const (
	Upgrade_0_1_30_PlanName = "v0.1.30"
)

// Upgrade_0_1_30 handles the upgrade to release `v0.1.30`.
// This upgrade includes:
// - Supplier service config update logic before activation fix
// - Experimental onchain metadata support
// - RelayMiner performance improvements (signatures, caching, etc)
// - New recovery wallets and docs
// See: https://github.com/pokt-network/poktroll/pull/1847
var Upgrade_0_1_30 = Upgrade{
	PlanName: Upgrade_0_1_30_PlanName,
	// No KVStore migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},

	// Upgrade Handler
	CreateUpgradeHandler: func(
		mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		// Ref: https://github.com/pokt-network/poktroll/compare/v0.1.29..v0.1.30

		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			// No state migrations are required for this upgrade because:
			// 1. Proto changes are backwards compatible:
			//    - Service.metadata is optional; existing services deserialize with nil metadata
			//    - ValidateBasic() explicitly allows nil metadata (x/shared/types/service.go:67)
			// 2. No new parameters were added (config.yml contains only test account changes)
			// 3. No new KVStore keys or module stores were introduced
			// 4. Logic fixes (IsSessionEndHeight, supplier config updates) are forward-looking:
			//    - They only affect new transactions and future state transitions
			//    - Existing on-chain state remains valid and compatible
			// 5. Query-only changes (supplier filters, service dehydration) don't affect consensus
			//
			// All changes are backwards compatible and require no data transformation.
			return vm, nil
		}
	},
}
