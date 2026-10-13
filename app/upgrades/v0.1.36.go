package upgrades

import (
	"context"

	cosmoslog "cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/pokt-network/poktroll/app/keepers"
	supplierkeeper "github.com/pokt-network/poktroll/x/supplier/keeper"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

const (
	Upgrade_0_1_36_PlanName = "v0.1.36"
)

// Upgrade_0_1_36 handles the upgrade to release `v0.1.36`.
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
//   - Supplier service cap (x/supplier). CONSENSUS-BREAKING (state): two new params,
//     min_stake_included_services = 1 and stake_per_additional_service = 0upokt
//     (cap disabled), a new Supplier.service_priority field written by every stake
//     message that lists services, a slash-path call that is a no-op while the cap
//     is disabled, and a param-change sweep. Behavior is unchanged until governance
//     sets a price. An upgrade handler that later changes min_stake,
//     min_stake_included_services or stake_per_additional_service with SetParams
//     MUST also call SupplierKeeper.ScheduleServiceCapSweep.
//   - MsgStakeSupplier can no longer lower a supplier's stake. CONSENSUS-BREAKING
//     (state): this reverts the immediate downstake (stake decrease returned to the
//     owner at once) that a lower stake at or above min_stake used to trigger.
//     Owners withdraw stake by unstaking, which goes through the unbonding period.
//     Morse supplier claims stake through the same path and never decrease stake.
//
// Gas changes (gas_used, hence LastResultsHash; no AppHash change):
//   - MsgCreateClaim / MsgSubmitProof: lower on memo hits.
//   - MsgUnstakeSupplier: reads the supplier dehydrated and its service config history
//     once (in BeginSupplierUnbonding) instead of hydrating it up front.
//   - MsgAddService: lower on a no-op update, higher when a card is set or changed.
//   - MsgUpdateParam / MsgUpdateParams (supplier): read the params an extra time,
//     and write the service cap sweep key when min_stake,
//     min_stake_included_services or stake_per_additional_service changes.
//   - MsgStakeSupplier and MsgClaimMorseSupplier (which stakes through
//     StakeSupplier): read the supplier params once more when the message lists
//     services (service cap check), and once less when the stake is unchanged (the
//     min stake re-check in the stake reconciliation is gone).
//   - Supplier records grow by service_priority on every stake message that lists
//     services, including new suppliers, so both those writes and later reads of
//     the records (claims, proofs, settlement) cost slightly more gas.
//
// Event changes (not hashed, but visible to indexers such as pocketdex):
//   - EventSupplierUnbondingBegin with reason BELOW_MIN_STAKE now reports the real
//     UnbondingEndHeight (unstake session end + unbonding period), not the unstake
//     session end height.
//   - New EventServiceMetadataUpdated.
//   - New EventSupplierServicesCapped.
//
// Operators: blocks before the upgrade height MUST be replayed with the previous
// binary (cosmovisor does this). The memo is not height-gated, so replaying old
// blocks with this binary yields different gas_used and a LastResultsHash mismatch.
var Upgrade_0_1_36 = Upgrade{
	PlanName: Upgrade_0_1_36_PlanName,
	// No KVStore migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},

	// Upgrade Handler
	CreateUpgradeHandler: func(
		mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		// Add new parameters by:
		// 1. Inspecting the diff between v0.1.35..v0.1.36
		// 2. Manually inspect changes in ignite's config.yml
		// 3. Update the upgrade handler here accordingly
		// Ref: https://github.com/pokt-network/poktroll/compare/v0.1.35..v0.1.36

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

			if err := seedServiceCapParams(ctx, logger, keepers.SupplierKeeper); err != nil {
				return vm, err
			}

			return vm, nil
		}
	},
}

// seedServiceCapParams writes the service cap params so they pass Validate().
// The cap ships disabled (price 0): behavior is unchanged until governance sets a price.
// No sweep is scheduled, since nothing can be over a disabled cap.
func seedServiceCapParams(ctx context.Context, logger cosmoslog.Logger, supplierKeeper supplierkeeper.Keeper) error {
	params := supplierKeeper.GetParams(ctx)
	params.MinStakeIncludedServices = suppliertypes.DefaultMinStakeIncludedServices
	price := suppliertypes.DefaultStakePerAdditionalService
	params.StakePerAdditionalService = &price

	if err := params.Validate(); err != nil {
		logger.Error("Invalid supplier params after seeding the service cap", "error", err)
		return err
	}
	if err := supplierKeeper.SetParams(ctx, params); err != nil {
		logger.Error("Failed to set supplier params", "error", err)
		return err
	}

	logger.Info("Seeded service cap params (disabled)", "params", params)
	return nil
}
