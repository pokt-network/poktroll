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
// - Fix for supplier service config update logic before activation (issue #1794)
//
// CONSENSUS-BREAKING (compute_units_per_relay pinned to session-start):
// Claim validation now reads a service's compute_units_per_relay (cupr) as it was
// effective at the claim's SESSION-START height instead of the live (claim-time)
// value (x/proof/keeper/service.go + msg_server_create_claim.go). Previously, changing
// a service's cupr while sessions were open forfeited every in-flight claim for that
// service with ErrProofComputeUnitsMismatch: the RelayMiner bakes the mine-time cupr
// into the append-only SMST, but the chain checked the new live cupr. Pinning to
// session-start makes both sides agree — a cupr change now only applies to sessions
// that START after it, mirroring how relay mining difficulty is already pinned.
//
// The at-height lookup is backed by a new cupr history store, written lazily on every
// AddService (x/service/keeper/service_compute_units_history.go): a create seeds the
// initial cupr, and a cupr change seeds the previous value (at height 1, covering all
// in-flight sessions) before recording the new value at the next session boundary.
//
// This requires NO KVStore migration or upgrade-handler logic:
//   - The new claim-check code path takes effect atomically at the upgrade height
//     when validators run the vNEXT binary (the vNEXT binary only ever processes
//     blocks >= H, so pre-upgrade claims — validated live — are never re-validated
//     under the new rule).
//   - cupr history is written lazily going forward; a service with no history falls
//     back to its current (deterministic) cupr, which equals the historical value
//     across the upgrade window because cupr changes are FROZEN during the binary
//     rollout (see docs/cupr_session_start_pin_plan.md "Sequencing").
//
// OPERATIONAL PREREQUISITE: freeze all cupr changes (MsgAddService updates that change
// compute_units_per_relay) from before this upgrade height until the RelayMiner fleet
// is upgraded to stamp session-start cupr. Unfreeze only afterwards.
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
