package upgrades

import (
	"context"

	cosmoslog "cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/pokt-network/poktroll/app/keepers"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

const (
	Upgrade_0_1_35_PlanName = "v0.1.35"
)

// Upgrade_0_1_35 handles the upgrade to release `v0.1.35`.
//
// This upgrade carries EIGHT consensus-breaking changes, all detailed below:
//  1. Settlement budget redistribution (head-split cap -> floor); ships as a no-op
//  2. Anti-collusion invariant enforced in tokenomics params validation
//  3. compute_units_per_relay pinned to the session-start height
//  4. Only the supplier owner can cancel an in-progress unstake (PR #1980)
//  5. Dead block-hash reads removed from the claim/proof commit-height calc (#1976)
//  6. Shared params validation rejects an all-zero claim/proof window offset set
//  7. AddService preserves stored service metadata when an update omits it
//  8. Gateway gains a metadata card, set via the new MsgUpdateGatewayMetadata
//
// CONSENSUS-BREAKING (shared params: zero-pending-sessions guard):
// x/shared Params.ValidateBasic now rejects a param set whose four claim/proof window
// offsets are all zero. Each offset is individually valid at zero, but their sum drives
// GetNumPendingSessions(), which settlement uses as a DIVISOR when deriving an
// application's per-session budget. math.Int.Quo panics on a zero divisor, and
// settlement runs in the EndBlocker — so the old validation admitted a governance
// param set that would have HALTED the chain at the next settlement block. A tx that
// previously passed validation now fails, hence consensus-breaking.
//
// This is fail-closed and requires no migration: it only rejects new writes. Live
// mainnet (11+10+1+10 = 32 blocks) and beta are unaffected, as are the defaults and
// every bulk-params artifact under tools/scripts/params/.
//
// CONSENSUS-BREAKING (service metadata preserved on update):
// MsgAddService no longer overwrites an existing service's metadata with nil when the
// message omits it (x/service/keeper/msg_server_add_service.go). MsgAddService is the
// ONLY update path for an existing service and always carries a full Service{}, so a
// message intending to change only compute_units_per_relay arrives with a nil Metadata;
// the old handler assigned it unconditionally and silently destroyed the stored value.
// The in-repo `pocketd tx service edit-service` command builds its message via
// NewMsgAddService, which never sets Metadata, so a routine batch cupr edit would have
// wiped it. A read-modify-write against a dehydrated query response had the same effect:
// both service queries strip metadata by setting it to nil (x/service/keeper/query_service.go),
// so re-submitting what was just read cleared it. The same txs now leave stored metadata
// intact, hence consensus-breaking.
//
// An update that DOES carry metadata still replaces it. No state migration: this only
// changes how future writes are applied. Mainnet exposure at the time of writing is one
// service (`ai-inference`, the only one with metadata set).
//
// KNOWN GAP: metadata can be ERASED but not set back to nil. An owner submits a minimal
// payload (e.g. `{}`, i.e. --card-base64 e30=) and it overwrites the
// stored blob, but Metadata.ValidateBasic rejects a zero-length card payload,
// so a non-nil Metadata always remains. Consumers must treat a non-nil Metadata as
// "may be empty", not "has content". Tracked as a TODO_TECHDEBT in the handler; a
// dedicated clear mechanism belongs with any rework of the metadata fields.
//
// CONSENSUS-BREAKING (gateway metadata card):
// `Gateway` gains a `metadata` field (field 4) of the same `pocket.shared.Metadata` type
// Service already uses, holding a gateway card: a small, self-describing JSON document
// (see docs/pocket_service_card.md). A new message, MsgUpdateGatewayMetadata, is the ONLY
// way to set it.
//
// The separate message is deliberate. MsgStakeGateway enforces a strictly-positive stake
// delta on every call (x/gateway/keeper/msg_server_stake_gateway.go, "MUST ALWAYS stake or
// upstake"), so folding the card into it would mean escrowing real POKT to fix a typo in a
// description. MsgStakeGateway is left completely untouched; a freshly staked gateway
// starts with no card and becomes descriptor-complete via a follow-up call.
//
// Semantics match MsgAddService: a nil metadata on the message leaves the stored card
// alone rather than clearing it, so a client that does not re-send the card cannot
// silently destroy it. The chain enforces the card's SIZE only -- it never parses the
// payload, schema-checks it, or attests to anything it claims. The gateway-exists check
// lives in the message server, not ValidateBasic, which is stateless.
//
// EVENTS: the four pre-existing gateway lifecycle events embed the whole `Gateway`, which
// now transitively includes the card. Those emission sites are changed to strip it
// (Gateway.DehydratedForEvent), so a card is never written into the event log on stake,
// unstake or unbond -- it can be up to MaxServiceMetadataSizeBytes and is always readable
// from state. The new EventGatewayMetadataUpdated carries primitives only (address,
// session end height, and the card's size in bytes), satisfying
// `make check_proto_event_fields`. Stripping is not a behavioural change relative to this
// release: no gateway carried a card before it.
//
// NO STATE MIGRATION and no upgrade-handler logic: `metadata` is a new optional field.
// Existing gateways decode with a nil card and every existing gateway operation is
// unaffected. The new message path takes effect atomically at the upgrade height.
//
// IDEMPOTENT WRITES: submitting a card byte-identical to the stored one -- or omitting it
// (nil metadata) -- succeeds without writing state or emitting an event. cosmos-sdk's
// KVStore marks a key dirty on ANY Set, so an identical re-Set still produces a fresh IAVL
// node at commit: a full extra copy of a card up to MaxServiceMetadataSizeBytes, retained
// forever by archive nodes. Without the short-circuit, rebroadcasting the same card is an
// unbounded state-growth lever costing only gas. This is query/state hygiene on a message
// that does not exist before this upgrade, so there is nothing to migrate.
//
// QUERIES: both gateway queries take a `dehydrated` flag (mirroring the service queries)
// that strips the card from the response. It defaults to FALSE -- a proto3 bool cannot
// default to true -- so cards are included unless asked otherwise. It matters most for
// enumeration: a card can be 256 KiB, so a default page of 100 carried cards is ~25 MB and
// would exceed the 4 MB default gRPC message limit. `pocketd query gateway list-gateway
// --dehydrated` sets it; `query gateway card <address>` reads an individual card. Adding a
// query request field is not consensus-breaking.
//
// NOT INCLUDED, deliberately: no service-id-filtered gateway query -- the advertised
// service list lives inside the opaque card, so the chain cannot filter on it; that belongs
// to an offchain index.
//
// CONSENSUS-BREAKING (supplier unstake cancellation):
// MsgStakeSupplier no longer cancels an in-progress unstake when sent by the operator;
// only the owner can. Previously an operator could re-stake to cancel the owner's
// unstake, locking the owner's funds in a supplier they had chosen to exit. Takes
// effect atomically at the upgrade height; no state migration.
//
// CONSENSUS-BREAKING (compute_units_per_relay pinned to session-start):
// Claim validation now reads a service's compute_units_per_relay (cupr) as it was
// effective at the claim's SESSION-START height instead of the live (claim-time)
// value (x/proof/keeper/service.go + msg_server_create_claim.go), and claim settlement
// checks the same pinned value (x/tokenomics/keeper/token_logic_modules.go). Previously,
// changing a service's cupr while sessions were open forfeited every in-flight claim for
// that service: the RelayMiner bakes the mine-time cupr into the append-only SMST, but
// the chain checked the new live cupr — rejecting the claim at creation with
// ErrProofComputeUnitsMismatch, or discarding it at settlement as a faulty claim.
// Pinning to session-start makes both sides agree — a cupr change now only applies to
// sessions that START after it, mirroring how relay mining difficulty is already pinned.
//
// The at-height lookup is backed by a new cupr history store, written lazily on every
// AddService (x/service/keeper/service_compute_units_history.go): a create seeds the
// initial cupr, and a cupr change seeds the previous value (at height 1, covering all
// in-flight sessions) before recording the new value at the next session boundary.
//
// This requires NO KVStore migration or upgrade-handler logic:
//   - The new claim-check code path takes effect atomically at the upgrade height
//     when validators run the v0.1.35 binary (the v0.1.35 binary only ever processes
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
//
// CONSENSUS-BREAKING (settlement budget redistribution):
// The per-supplier head-split cap (appStake / numPendingSessions / actualNumSuppliers)
// becomes a guaranteed FLOOR rather than a hard ceiling. Budget left unused by idle or
// light suppliers can be redistributed to suppliers that served above their floor,
// bounded by the application's committed per-session budget B. Introduces the new
// tokenomics param `overservicing_bonus_multiplier`, seeded to 1 by the handler below so
// the consensus change SHIPS AS AN EXACT NO-OP: m == 1 reproduces the legacy head-split
// cap byte-for-byte, and the zero value is coerced to 1 at read time so an unset or
// clobbered param can never silently enable redistribution. Governance opens
// redistribution afterwards by raising the multiplier — no second upgrade required.
//
// Also introduces the anti-collusion invariant
// `mint_ratio * mint_equals_burn_claim_distribution.supplier < 1`, REPORTED as a warning
// rather than enforced. The head-split cap was incidentally bounding collusion throughput;
// once it is demoted to a floor, this round-trip factor becomes the primary signal for
// application/supplier self-dealing. It is not a hard validation error because the
// distribution is DAO-governed, the product is <= 1 for every legal param set (so
// collusion can never be profitable, only break-even), and failing here would run inside
// consensus and halt the chain at the upgrade height. See
// Params.CheckAntiCollusionInvariant.
//
// CONSENSUS-BREAKING (gas: dead block-hash reads removed, #1976):
// GetEarliestSupplierClaimCommitHeight / GetEarliestSupplierProofCommitHeight ignore
// their block-hash argument (distribution seeding is disabled), but the on-chain callers
// were still issuing a gas-metered GetBlockHash store read for the discarded value
// (x/proof/types/shared_query_client.go, x/proof/keeper/msg_server_submit_proof.go).
// They now pass nil. This CHANGES GAS on the MsgCreateClaim and MsgSubmitProof paths —
// it takes effect atomically at the upgrade height and needs no migration, but a node
// still running an older binary past that height will diverge on gas_used (and hence
// LastResultsHash) even though its AppHash matches. The proof-requirement seed itself is
// unchanged; only the discarded window-open reads were removed.
var Upgrade_0_1_35 = Upgrade{
	PlanName: Upgrade_0_1_35_PlanName,
	// No KVStore migrations in this upgrade.
	StoreUpgrades: storetypes.StoreUpgrades{},

	// Upgrade Handler
	CreateUpgradeHandler: func(
		mm *module.Manager,
		keepers *keepers.Keepers,
		configurator module.Configurator,
	) upgradetypes.UpgradeHandler {
		// Add new parameters by:
		// 1. Inspecting the diff between v0.1.34..v0.1.35
		// 2. Manually inspect changes in ignite's config.yml
		// 3. Update the upgrade handler here accordingly
		// Ref: https://github.com/pokt-network/poktroll/compare/v0.1.34..v0.1.35

		// Seed the new overservicing_bonus_multiplier tokenomics param.
		//
		// On upgrade the previously-stored tokenomics params deserialize with the new field at
		// its proto3 zero value (0). The settlement path already treats 0 as 1 (the legacy
		// floor cap), so this upgrade is economically inert even without any migration — the
		// zero-value is benign by design. We still set it to 1 explicitly so the stored param
		// is self-documenting rather than relying on the read-time coercion. Governance later
		// raises the multiplier to enable redistribution — no second upgrade required.
		applyNewParameters := func(ctx context.Context, logger cosmoslog.Logger) (err error) {
			logger.Info("Starting settlement budget redistribution parameter updates",
				"upgrade_plan_name", Upgrade_0_1_35_PlanName)

			tokenomicsParams := keepers.TokenomicsKeeper.GetParams(ctx)

			// Set overservicing_bonus_multiplier to its default (1 = no-op / legacy
			// head-split cap) if unset, so redistribution stays OFF until governance opts in.
			if tokenomicsParams.OverservicingBonusMultiplier == 0 {
				tokenomicsParams.OverservicingBonusMultiplier = tokenomicstypes.DefaultOverservicingBonusMultiplier
				logger.Info("Setting default overservicing_bonus_multiplier to 1 (no-op; governance can enable redistribution)")
			}

			// Ensure the new parameter set is valid.
			if err = tokenomicsParams.ValidateBasic(); err != nil {
				logger.Error("Failed to validate tokenomics params", "error", err)
				return err
			}

			// Report — but do NOT fail on — an anti-collusion invariant violation.
			// Returning an error here runs inside consensus and would halt the chain at
			// the upgrade height over a DAO-governed policy value that can, at worst,
			// make self-dealing break-even. See Params.CheckAntiCollusionInvariant.
			tokenomicsParams.LogAntiCollusionInvariantViolation(logger)

			if err = keepers.TokenomicsKeeper.SetParams(ctx, tokenomicsParams); err != nil {
				logger.Error("Failed to set tokenomics params", "error", err)
				return err
			}
			logger.Info("Successfully seeded overservicing_bonus_multiplier", "new_params", tokenomicsParams)

			return nil
		}

		return func(ctx context.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
			sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
			logger := sdkCtx.Logger()

			if err := applyNewParameters(ctx, logger); err != nil {
				return vm, err
			}

			return vm, nil
		}
	},
}
