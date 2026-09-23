package upgrades

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/pokt-network/poktroll/app/keepers"
)

// TODO_NEXT_UPGRADE: Rename NEXT with the appropriate next
// upgrade version number and update comment versions.

const (
	Upgrade_NEXT_PlanName = "vNEXT"
)

// Upgrade_NEXT handles the upgrade to release `vNEXT`.
// This upgrade adds:
//   - Block-scoped session supplier memo (x/session transient store, see
//     x/session/keeper/session_memo.go). CONSENSUS-BREAKING: memo hits lower
//     gas_used of claim/proof txs, which changes LastResultsHash (not AppHash).
//     It MUST only ship through this upgrade, never in a patch release. Needs no
//     handler code or StoreUpgrades: transient stores hold no committed state.
//   - Supplier unbonding fixes (x/supplier BeginSupplierUnbonding). CONSENSUS-BREAKING
//     (state): a supplier slashed below min stake during settlement is now persisted
//     with its service config and unstaking indexes, so it leaves sessions from the
//     next one on and is unbonded (it previously stayed selectable and never
//     unbonded); unstaking no longer pushes an already-deactivated service config's
//     deactivation later (which reactivated it for past sessions). No migration: no
//     mainnet supplier was below min stake or had overlapping configs (2026-09-19).
var Upgrade_NEXT = Upgrade{
	PlanName: Upgrade_NEXT_PlanName,
	// No KVStore migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},

	// Upgrade Handler
	CreateUpgradeHandler: func(
		mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		// Add new parameters by:
		// 1. Inspecting the diff between vPREV..vNEXT
		// 2. Manually inspect changes in ignite's config.yml
		// 3. Update the upgrade handler here accordingly
		// Ref: https://github.com/pokt-network/poktroll/compare/vPREV..vNEXT

		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			return vm, nil
		}
	},
}
