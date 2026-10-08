package upgrades

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
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
//     gas_used of claim/proof txs, which changes LastResultsHash. It does not
//     change the AppHash while block max_gas is -1 (mainnet today); under a block
//     gas limit it could. It MUST only ship through this upgrade, never in a patch
//     release. Needs no handler code or StoreUpgrades: transient stores hold no
//     committed state.
//   - Supplier unbonding fixes (x/supplier BeginSupplierUnbonding). CONSENSUS-BREAKING
//     (state): a supplier slashed below min stake during settlement is now persisted
//     with its service config and unstaking indexes, so it leaves sessions from the
//     next one on and is unbonded (it previously stayed selectable and never
//     unbonded); unstaking no longer pushes an already-deactivated service config's
//     deactivation later (which reactivated it for past sessions).
//   - One-shot repair (RepairUnindexedUnbondingSuppliers): every supplier the old
//     settlement code left unbonding but unindexed starts unbonding again from
//     this session, so it leaves sessions and is unbonded. Mainnet had none on
//     2026-10-07, but two suppliers sat at or 1 upokt above min stake with a
//     1 upokt proof_missing_penalty, so one could get stuck before the upgrade.
//     Testnets were not checked; the handler logs every repaired address.
//   - EventServiceMetadataUpdated (x/service, #2009): MsgAddService emits it whenever
//     the stored card changes, including a service created with a card. CONSENSUS-
//     BREAKING, state and gas: an update that changes nothing (same name, cupr and
//     card) now skips the store write (as UpdateGatewayMetadata already does), and
//     emitting the event reads the shared params (~1.1k gas per card set or change).
//     Do NOT backport the emission alone as a patch: the extra read changes gas_used.
//     No handler code needed.
//
// Gas changes (gas_used, hence LastResultsHash; no AppHash change):
//   - MsgCreateClaim / MsgSubmitProof: lower on memo hits.
//   - MsgUnstakeSupplier: reads the supplier dehydrated and its service config history
//     once (in BeginSupplierUnbonding) instead of hydrating it up front.
//   - MsgAddService: lower on a no-op update, higher when a card is set or changed.
//
// Event changes (not hashed, but visible to indexers such as pocketdex):
//   - EventSupplierUnbondingBegin with reason BELOW_MIN_STAKE now reports the real
//     UnbondingEndHeight (unstake session end + unbonding period), not the unstake
//     session end height.
//   - New EventServiceMetadataUpdated.
//
// Operators: blocks before the upgrade height MUST be replayed with the previous
// binary (cosmovisor does this). The memo is not height-gated, so replaying old
// blocks with this binary yields different gas_used and a LastResultsHash mismatch.
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
			logger := cosmostypes.UnwrapSDKContext(ctx).Logger()

			// Repair suppliers force-unstaked below min stake by the old settlement
			// code, which never indexed them. Never errors: a failure here would
			// halt the chain at the upgrade height.
			repaired := keepers.SupplierKeeper.RepairUnindexedUnbondingSuppliers(ctx)
			for _, operatorAddress := range repaired {
				logger.Info("Repaired unindexed unbonding supplier", "operator_address", operatorAddress)
			}
			logger.Info("Repaired unindexed unbonding suppliers", "count", len(repaired))

			return vm, nil
		}
	},
}
