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
//     The operator may now cancel an unbonding that settlement (or the repair
//     below) started because the stake fell below min_stake, by staking again with
//     its services and at least min_stake, paid by the operator. Such unbondings are
//     recorded under a new BelowMinStakeUnbonding/ store prefix in x/supplier,
//     cleared on cancel and on removal (one more read on every unbonding cancel, one
//     more delete on a recorded one). Any other unbonding, including an owner's
//     unstake whose stake later falls below min_stake, can still be cancelled only
//     by the owner. An owner's stake-only top-up still cancels any unbonding, but leaves
//     the deactivated services deactivated: the operator must stake again with
//     them. Before, the settlement path deactivated nothing, so a top-up alone
//     restored the supplier. CONSENSUS-BREAKING (state).
//   - One-shot repair (RepairUnindexedUnbondingSuppliers): every supplier the old
//     settlement code left unbonding but unindexed starts unbonding again from
//     this session, so it leaves sessions and is unbonded. Mainnet had none on
//     2026-10-07, but two suppliers sat at or 1 upokt above min stake with a
//     1 upokt proof_missing_penalty, so one could get stuck before the upgrade.
//     Each repaired supplier gets a new EventSupplierUnbondingBegin with its new
//     heights. Testnets were not checked; the handler logs every repaired address.
//   - Settlement resolves a claim's supplier service config (revenue share) at the
//     claim's session start, not at the settlement height. CONSENSUS-BREAKING
//     (state): a claim for a session the supplier served now settles even when the
//     service was deactivated since (unstake, slash below min stake, service cap);
//     it was previously discarded. Claims of one supplier and service from
//     different sessions settling in the same block each use their own session's
//     config. A claim whose token logic modules fail leaves the application stake
//     and the validator rewards of the settlement unchanged.
//   - Deactivated supplier service configs are kept until the proof window of
//     every session they served has closed, under the params of each session,
//     including after a params change that shortens the claim and proof windows.
//     CONSENSUS-BREAKING (state): configs can be pruned later than before.
//   - MsgCreateClaim and MsgSubmitProof: the session header's start and end
//     heights must equal the onchain session's. CONSENSUS-BREAKING (state).
//     RelayMiners already reject relays whose header differs from the onchain one.
//   - Claim compute units are compared with the relay count times the compute
//     units per relay as a full-width product, at MsgCreateClaim and at settlement.
//     CONSENSUS-BREAKING (state): a claim whose product exceeds uint64 is rejected,
//     and such a claim created before the upgrade is discarded at settlement.
//   - x/shared: proof_window_open_offset_blocks must be at least 1 (defaults now
//     proof window open 1 / close 3, same total). No handler code: mainnet and Beta
//     are at 1. A chain still at 0 keeps working, but its shared param updates are
//     rejected until they set it to at least 1.
//   - Proof validation recovers from a panic only while decoding and verifying the
//     submitted relay and proof (reported as an invalid proof); chain state reads,
//     including the application ring lookup, run outside it, so a panic there stops
//     the node as before. The one exception is unchanged from v0.1.35: x/proof's
//     account reads recover a panic (only possible for an account that cannot be
//     decoded) and report it as an invalid proof; it is now logged at error level.
//     CONSENSUS-BREAKING (state): relay request ring
//     signatures must have between 2 and 64 members (MaxRingSignatureMembers, a
//     fixed protocol constant), and a size that matches the member count. A
//     signature may not have more members than the application's expected ring,
//     counted as signers build it: the application, its delegated gateways and the
//     gateways undelegated since the session, a key listed twice counted twice (the
//     application key twice when it has no gateways). SDKs and gateways building
//     rings must not drop repeated keys. The expected ring is now read at the
//     session end height of the relay's header, not at the session end computed
//     from the live session length. RelayMiners compare the relay's session header
//     with the onchain session before checking its signature.
//     The expected ring leaves out a gateway whose account has no public key or a
//     key that is not secp256k1 (no signer can use it); the application's own key
//     is still required. CONSENSUS-BREAKING (state, gas: MsgDelegateToGateway reads
//     the gateway account). MsgDelegateToGateway rejects a
//     gateway whose account key is not secp256k1 (one with no key yet is accepted).
//     SDKs and gateways building rings should leave such gateways out too.
//   - x/application: max_delegated_gateways must be at most 31
//     (MaxDelegatedGatewaysLimit; mainnet uses 7). A transfer into an existing
//     application fails, with EventTransferError, when the merged delegations
//     would exceed max_delegated_gateways. MsgDelegateToGateway, and a transfer
//     merge, are also rejected when the application's ring (the application, its
//     delegated gateways and the gateways undelegated in the current session)
//     would exceed 64 members. MsgDelegateToGateway reads the shared params once
//     more for it (gas). CONSENSUS-BREAKING (state). No handler
//     code needed.
//   - Settlement no longer moves the unbonding height of an application that is
//     already unbonding when its stake is below the minimum. CONSENSUS-BREAKING
//     (state).
//   - EventServiceMetadataUpdated (x/service, #2009): MsgAddService emits it whenever
//     the stored card changes, including a service created with a card. CONSENSUS-
//     BREAKING, state and gas: an update that changes nothing (same name, cupr and
//     card) now skips the store write (as UpdateGatewayMetadata already does), and
//     emitting the event reads the shared params (~1.1k gas per card set or change).
//     Do NOT backport the emission alone as a patch: the extra read changes gas_used.
//     No handler code needed.
//   - Supplier service cap (x/supplier). CONSENSUS-BREAKING (state): two new params,
//     min_stake_included_services = 1 and stake_per_additional_service = 0upokt
//     (cap disabled), a new ServicePriority/ store prefix in x/supplier (each
//     supplier's service order, written by every stake message that lists services;
//     not on the Supplier record, which claims and proofs read), a slash-path call
//     that is a no-op while the cap is disabled, and a param-change sweep. Behavior
//     is unchanged until governance sets a price. SetParams schedules the sweep
//     whenever the cap is enabled and a cap param changes, including from genesis
//     and upgrade handlers. New store prefixes in an existing store need no
//     StoreUpgrades.
//     A min_stake_included_services of 0 in state (never accepted by validation)
//     counts as 1, so the cap can never allow a supplier no services.
//     Cap deactivations use the shared params effective at the current height,
//     so at the block a new session length takes effect they follow the new grid.
//   - MsgStakeSupplier can no longer lower a supplier's stake. CONSENSUS-BREAKING
//     (state): this reverts the immediate downstake (stake decrease returned to the
//     owner at once) that a lower stake at or above min_stake used to trigger.
//     Owners withdraw stake by unstaking, which goes through the unbonding period.
//     Morse supplier claims stake through the same path and never decrease stake.
//   - Settlement pays a module to account transfer whose recipient the bank module
//     refuses (its blocked addresses: fee collector, distribution, mint, bonded and
//     not bonded pools) to the DAO reward address instead, or leaves it in the
//     sender module when that address is blocked too, and logs it. CONSENSUS-
//     BREAKING (state). MsgStakeSupplier rejects such an address as a revenue share
//     recipient, and MsgTransferService as the new service owner. CONSENSUS-BREAKING
//     (state, gas: one bank lookup per address).
//   - Application per_session_spend_limit history (new Application field 10,
//     per_session_spend_limit_history). A spend limit change takes effect at the
//     next session start, like service config changes, and settlement applies the
//     limit in effect at the claim's session start. A change replaces every change
//     still pending. A nil limit amount is rejected instead of panicking in
//     validation. CONSENSUS-BREAKING (state):
//     stake and transfer messages that change the limit write a history entry, and
//     application records and events carry it. An application with no history
//     (every application at the upgrade) uses its current limit. No handler code.
//   - Undelegated gateways stay in an application's pending undelegations at least
//     until the proof window of their session closes, also when the claim and proof
//     windows are longer than the grace period plus two sessions. Each entry's
//     retention uses the shared params in effect at its session end, as proof
//     validation does. CONSENSUS-BREAKING
//     (state); unchanged at mainnet and Beta params (50 blocks kept, proof windows
//     close 33 blocks after the session end).
//   - x/shared: the supplier and application unbonding periods must last at least
//     until the proof window close height, which is one block later than validation
//     counted. No handler code: mainnet and Beta are well above it; LocalNet and
//     Alpha are exactly at it. The default supplier and application unbonding
//     periods are now 2 sessions (new chains only).
//
// Gas changes (gas_used, hence LastResultsHash; no AppHash change):
//   - MsgCreateClaim / MsgSubmitProof: lower on memo hits.
//   - MsgUnstakeSupplier: reads the supplier dehydrated and its service config history
//     once (in BeginSupplierUnbonding), and rewrites only the service configs whose
//     deactivation height changes instead of re-indexing every config (lower).
//     The below-min-stake unbonding in settlement and the one-shot repair use the
//     same path. Its EventSupplierUnbondingBegin now reads the params history once
//     for the unbonding end height (higher).
//   - MsgUnstakeApplication / MsgUnstakeGateway: the unbonding begin event reads the
//     params history once for the unbonding end height instead of the live params.
//     Do NOT backport these event changes alone as a patch: the reads change gas_used.
//   - MsgStakeApplication / MsgTransferApplication: no shared params read when the
//     spend limit does not change (lower).
//   - MsgAddService: lower on a no-op update, higher when a card is set or changed.
//   - MsgUpdateParam / MsgUpdateParams (supplier): write the service cap sweep key
//     when the cap is enabled and min_stake, min_stake_included_services or
//     stake_per_additional_service changes.
//   - MsgStakeSupplier and MsgClaimMorseSupplier (which stakes through
//     StakeSupplier): read the supplier params once instead of twice; a message
//     that lists services reads the service priority and writes it only when the
//     order changes. The
//     min stake re-check in the stake reconciliation is gone. A message without
//     services (stake top-up, unbonding cancel) no longer re-indexes the service
//     config history (much lower for suppliers with many services).
//
// Event changes (not hashed, but visible to indexers such as pocketdex):
//   - EventSupplierUnbondingBegin with reason BELOW_MIN_STAKE now reports the real
//     UnbondingEndHeight (unstake session end + unbonding period, with the params
//     at the unstake session end, as the unbonding itself uses), not the unstake
//     session end height. The VOLUNTARY reason and EventApplicationUnbondingBegin
//     with reason BELOW_MIN_STAKE also use the params at the unstake session end.
//     The repair's EventSupplierUnbondingBegin carries the dehydrated supplier
//     (no services), like the VOLUNTARY one; the settlement one carries the
//     supplier cached for the settlement batch, with the first service config seen
//     per service across that supplier's claims in the block.
//   - EventApplicationUnbondingBegin (ELECTIVE) and EventGatewayUnbondingBegin
//     report the unbonding end height with the params at the unstake session end,
//     the height at which the unbonding EndBlockers release them.
//   - Events emitted while settling a claim are dropped when its token logic
//     modules fail and the claim is discarded (EventClaimDiscarded remains).
//   - New EventServiceMetadataUpdated.
//   - New EventSupplierServicesCapped, one per deactivated service, primitives only.
//   - EventSupplierServiceConfigActivated is no longer emitted for a config that is
//     deactivated at or before its activation height (it never became active).
//
// CometBFT (pokt-network fork, upstream v0.38.21 -> v0.38.26): validators MUST
// check `double_sign_check_height` in config.toml before the upgrade. A value of 1
// was a no-op before and now checks the previous block, so a validator restarting
// at the upgrade height with 1 finds its own signature and refuses to start. Set it
// to 0 (the default) for the upgrade restart. Other engine changes: the new
// `event_bus_buffer_capacity` setting (0, the default, keeps the previous
// unbuffered event bus), stricter blocksync response validation, a mempool message
// size filter, and a 256-character limit on the consensus params authority field.
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
