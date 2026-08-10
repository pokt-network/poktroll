> ⚠️ **Consensus-breaking release.** Requires a coordinated network upgrade via the `v0.1.35` upgrade handler — all validators and full nodes must run this binary at the upgrade height.
>
> ⚠️ **Operational prerequisite:** FREEZE all `compute_units_per_relay` changes (`MsgAddService` updates that change cupr) from before the upgrade height until the RelayMiner fleet is upgraded. Unfreeze only afterwards. See "Operator & Governance Notes".

## Highlights

### 💸 Settlement budget redistribution — head-split cap becomes a floor (consensus-breaking, ships as a NO-OP)

The per-supplier head-split (`appStake / numPendingSessions / actualNumSuppliers`) is demoted from a hard **ceiling** to a guaranteed **floor**. Budget left unused by idle or light suppliers can now be redistributed to suppliers that served above their floor, in proportion to each one's excess, bounded by the application's committed per-session budget `B`.

This addresses measured **delivered-but-unpaid work**: 24–31% of all claimed value on mainnet, amplified ~15× by the N=60→20 session-length flip.

- New tokenomics param **`overservicing_bonus_multiplier`** (field 10), seeded to `1` by the upgrade handler.
- **`m == 1` reproduces the legacy head-split cap byte-for-byte.** The change ships economically inert; governance opens redistribution afterwards by raising the multiplier — no second upgrade required.
- The **zero value is coerced to `1` at read time**, not treated as "unlimited", so an unset, un-migrated, or clobbered param can never silently enable redistribution.
- Bounded above by `MaxOverservicingBonusMultiplier` (1000). Any value at or above `num_suppliers_per_session` already removes the cap in practice (the application budget `B` binds first), so a larger number buys nothing — rejecting it turns a fat-fingered extra digit into a loud validation error instead of a silent no-op.
- Solvency invariant: `Σ floor + Σ bonus ≤ N × floor = B`, so an application's stake can never be overdrawn.

### 🔒 Anti-collusion invariant now enforced (consensus-breaking)

`mint_ratio × mint_equals_burn_claim_distribution.supplier < 1` is now a hard validation error in tokenomics params. The head-split cap was incidentally bounding collusion throughput; once demoted to a floor, this invariant becomes the primary guard against application/supplier self-dealing round-trips. Current mainnet values (`0.975 × 0.79 = 0.770`) satisfy it.

The check is computed over `big.Rat` via `encoding.Float64ToRat`, not float64: it runs in the upgrade handler and on every params update, so it follows the repo's rule for float64 params. This also closes a hole — a non-finite `supplier` share made the float comparison `NaN >= 1` evaluate false and slip through; the `big.Rat` conversion rejects it outright.

### 🧾 Governance bulk-params files backfilled

`tools/scripts/params/bulk_params_{main,beta,alpha}/tokenomics_params.json` were stale: all three omitted `mint_ratio`, and main/beta carried an outdated `mint_equals_burn_claim_distribution`. Since `MsgUpdateParams` replaces the whole struct, running one would have clobbered live values. Main and beta are now backfilled from **verified live chain state**; a new test (`TestBulkParamsJSON_RoundTripsAndValidates`) decodes every bulk file through the real proto-JSON codec, runs `ValidateBasic`, and fails if any tokenomics param is missing — so this cannot silently regress when the next param is added.

### 📌 `compute_units_per_relay` pinned to session-start (consensus-breaking)

Changing a service's cupr mid-session used to forfeit **every in-flight claim** for that service: the RelayMiner bakes the mine-time cupr into the append-only SMST, but the chain checked the new live value — rejecting the claim with `ErrProofComputeUnitsMismatch` at creation, or discarding it as faulty at settlement. On 2026-08-03 a single tx carrying 59 `MsgAddService` updates wiped ~90% of all network claims for two sessions.

Claim validation and settlement now both read the cupr that was effective at the claim's **session-start** height, mirroring how relay mining difficulty is already pinned. A cupr change applies only to sessions that START after it.

- New cupr history store, written lazily on every `AddService` (create seeds the initial value; a change seeds the previous value before recording the new one at the next session boundary).
- New queries: `service/ComputeUnitsPerRelayAtHeight` and `service/ComputeUnitsPerRelayHistory` (+ autocli commands).
- RelayMiner now stamps relays with the session-start cupr via the new query, so both sides agree.
- **No KVStore migration.** A service with no recorded history falls back to its current (deterministic) cupr, which equals the historical value across the upgrade window *because cupr changes are frozen during the binary rollout*.

### 🔑 Only the owner can cancel an in-progress unstake (consensus-breaking)

`MsgStakeSupplier` no longer cancels an in-progress unbonding when sent by the **operator**. Previously an operator could repeatedly re-stake (paying only the staking fee) to cancel an owner-initiated unstake, locking the owner's escrowed stake in a supplier they had chosen to exit. Takes effect atomically at the upgrade height; no state migration.

### 🧹 Dead block-hash reads removed from the claim/proof path (consensus-breaking: gas)

`GetEarliestSupplierClaimCommitHeight` / `GetEarliestSupplierProofCommitHeight` ignore their block-hash argument (distribution seeding is disabled), yet the on-chain callers were still issuing a gas-metered `GetBlockHash` store read for the discarded value. A discarded read that still consumes consensus gas is a latent nondeterminism surface — if it ever returned a different byte length across nodes, `gas_used` would diverge and `LastResultsHash` would split while `AppHash` stayed identical. That is the signature of the beta-lego block-432943 halt (suspected carrier, not a proven root cause). These reads now pass `nil`.

**This changes gas on the `MsgCreateClaim` and `MsgSubmitProof` paths.** The proof-requirement seed itself is unchanged.

The RelayMiner's off-chain twin (`pkg/client/query/sharedquerier.go`) did the same thing over RPC — a full `blockQuerier.Block` call per claim/proof window for a discarded hash, which also turned any RPC hiccup into an error return. That fetch is gone too, along with its now-unused block-hash cache.

### 🛑 Shared params: reject an all-zero claim/proof window offset set (consensus-breaking)

Each of the four claim/proof window offsets is individually valid at zero, but their **sum** drives `GetNumPendingSessions()` — which settlement uses as a **divisor** to derive an application's per-session budget. `math.Int.Quo` panics on a zero divisor, and settlement runs in the EndBlocker, so the old validation admitted a governance param set that would have **halted the chain** at the next settlement block.

`x/shared` `Params.ValidateBasic` now rejects that combination. Fail-closed, no migration — it only rejects new writes. Live mainnet (`11+10+1+10 = 32`), beta, the defaults, and every artifact under `tools/scripts/params/` are unaffected.

### 🗂️ Service metadata no longer wiped by an unrelated update (consensus-breaking)

`MsgAddService` is the **only** update path for an existing service and always carries a full `Service{}`, so a message intending to change only `compute_units_per_relay` arrives with a nil `Metadata`. The handler assigned it unconditionally, silently destroying the stored value.

This was reachable through our own tooling: `pocketd tx service edit-service` (the batch cupr editor) builds its message via `NewMsgAddService`, which never sets `Metadata` — so a routine cupr edit would have wiped the service's metadata. A read-modify-write against a **dehydrated** query response did the same, since both service queries strip metadata by setting it to nil. The handler now only overwrites metadata when the message actually carries it; an update that does supply metadata still replaces it. No migration.

**Known gap:** metadata can be *erased* but not set back to nil. Submitting a minimal payload — e.g. `pocketd tx service add-service ... --card-base64 e30=` (`{}`) — overwrites the stored blob, but `Metadata.ValidateBasic` rejects a zero-length card payload, so a non-nil `Metadata` always remains. Consumers should treat a non-nil `Metadata` as *may be empty*, not *has content*. A dedicated clear mechanism belongs with any rework of the metadata fields (tracked as a `TODO_TECHDEBT` in the handler).

### ✅ `validate-card` — catch a malformed card before it costs gas

The chain enforces a card's **size only** and never parses the payload, so client-side validation is the only place a malformed card can be caught before it lands onchain. Two new offline commands do that:

```bash
pocketd tx service validate-card ./card.json
pocketd tx gateway validate-card ./gateway-card.json
```

`add-service` and `update-gateway-metadata` run the same check automatically before broadcasting; `--skip-card-validation` opts out for anyone deliberately storing something that is not a card.

Every violation is reported at once, with the offending JSON path:

```
card does not match the service card schema:
  /rpc_types/0/type: value must be one of 'GRPC', 'WEBSOCKET', 'JSON_RPC', 'REST', 'COMET_BFT'
  /specs/0/sha256: 'NOTAHASH' does not match pattern '^[0-9a-f]{64}$'
```

The canonical schemas now live in `pkg/cards/` and are compiled into `pocketd` — nothing is fetched over the network. `docs/pocket_service_card.md` is the prose companion, not a second copy.

### 🪪 Gateway cards via `MsgUpdateGatewayMetadata` (consensus-breaking)

Gateways had no onchain identity beyond `address` and `stake`, so tooling hardcoded names and endpoints per gateway. `Gateway` now carries `metadata` (field 4) — the same `pocket.shared.Metadata` container Service uses — holding a **gateway card**: a small, self-describing JSON document.

The card is set **only** by a new `MsgUpdateGatewayMetadata`, never by `MsgStakeGateway`. That separation is the point: `MsgStakeGateway` enforces a strictly-positive stake delta on every call, so folding metadata into it would mean escrowing real POKT to fix a typo in a description. `MsgStakeGateway` is untouched — a freshly staked gateway starts with no card and becomes descriptor-complete via a follow-up call, and updating a card costs gas only.

Semantics match `MsgAddService`: a nil `metadata` leaves the stored card alone rather than clearing it. The chain enforces **size only** — it never parses the payload or attests to anything it claims.

**Idempotent writes:** submitting a card byte-identical to the stored one — or omitting it — succeeds but writes nothing and emits no event. A cosmos-sdk KVStore marks a key dirty on *any* `Set`, so an identical re-`Set` still produces a fresh IAVL node at commit: another full copy of a card up to 256 KiB, kept forever by archive nodes. Without this, rebroadcasting the same card would be an unbounded state-growth lever costing only gas.

**Events:** the four pre-existing gateway lifecycle events embed the whole `Gateway`, which now transitively includes the card. Left alone, every stake/unstake/unbond would write up to 256 KiB into the event log for a payload that never changes on those paths and is always readable from state. Those emission sites now strip it (mirroring the `dehydrated` flag on the service queries), and the new `EventGatewayMetadataUpdated` carries primitives only: address, session end height, and the card's size in bytes.

**No state migration.** `metadata` is a new optional field; existing gateways decode with a nil card and every existing gateway operation is unaffected.

**Queries:** both gateway queries now take a `dehydrated` flag that strips the card from the response, mirroring the service queries. It defaults to `false` (cards included) — a proto3 bool cannot default to `true`. It matters most for enumeration: a card can be 256 KiB, so a default page of 100 carried cards is ~25 MB and would exceed the 4 MB default gRPC message limit.

```bash
pocketd query gateway list-gateway --dehydrated   # omit every card
pocketd query gateway show-gateway pokt1... --dehydrated
pocketd query gateway card pokt1...               # read one card, decoded
```

Adding a query request field is not consensus-breaking.

Not included, deliberately: no service-id-filtered gateway query — the advertised service list lives inside the opaque card, so the chain cannot filter on it; that belongs to an offchain index.

Schema: `docs/pocket_gateway_card.schema.json`, documented alongside the service card in `docs/pocket_service_card.md`.

### 🏷️ `experimental_api_specs` renamed to `card` (client-facing, NOT consensus-breaking)

`Service.metadata`'s only field is renamed `experimental_api_specs` → `card`, and the message it lives in is redocumented as a generic descriptor container. The field **number is unchanged**, so the wire format, the stored state, and every hash are byte-identical — this is not a consensus change and needs no upgrade handler.

What does change is the **JSON key** returned by `Service(id)` and `AllServices`:

```diff
- "metadata": {"experimental_api_specs": "eyJzY2hlbWEi…"}
+ "metadata": {"card": "eyJzY2hlbWEi…"}
```

Any off-chain consumer parsing that key must be updated. Done now because exactly one mainnet service currently sets the field, so this is the cheapest it will ever be.

CLI flags follow: `--card-base64` / `--card-file`. The old `--experimental-metadata-base64` / `--experimental-metadata-file` remain as **deprecated aliases**, so existing scripts keep working; passing both a flag and its alias is an error rather than a silent precedence rule.

The field now carries a **service card** — a small, self-describing JSON document. See `docs/pocket_service_card.md` for the schema and `docs/pocket_service_card.schema.json` to validate against.

### ⚡ RelayMiner reliability (off-chain, backward-compatible)

- **Claim/proof tx re-broadcast.** A claim/proof tx was broadcast exactly once; if evicted from the mempool before inclusion it was never re-injected and the claim was forfeited at window close (`PROOF_MISSING`) even when later blocks in the window were empty. Pending txs are now re-broadcast up to twice, spread across the window with a deterministic per-tx jitter, never within the safety margin of the timeout height. Waves are bounded (`maxTxRebroadcastsPerBlock`), drained on a worker pool **off the committed-block goroutine**, and never overlap — so a large fleet's batch cannot delay timeout processing or stall block consumption.
- **WebSocket goroutine leak fixed.** `stopChan` was never closed, leaking a `goPublish` goroutine (plus the observable and its buffer) per connection; a client that disconnected early kept the bridge's goroutines and its block subscription pinned until `closeHeight`. Bridges now tear down on the first stop signal, after confirming every `stopChan` sender has returned.
- **Hijacked-connection write fixed.** After a WebSocket upgrade, the HTTP response writer is dead; error paths no longer write to it (`http: response.Write on hijacked connection`) and close the hijacked conn themselves instead of leaking the fd.
- **zerolog Event reuse panic fixed.** `error_reply.go` stored a polylog `Event` and logged through it twice, returning the same `*zerolog.Event` to the `sync.Pool` twice — handing one event to two goroutines and crashing the process with `index out of range [-1]`, typically surfacing in an unrelated (websocket) goroutine. Hold the `Logger`, never the `Event`.
- **Cache clearing decoupled from the shared-params cache.** Every clear handler gated on reading the shared-params cache, but that cache clears *itself* through the same fan-out — so handlers that lost the race observed an empty cache and skipped their own clear, and nothing repopulated it. Handlers now keep a local copy. Clearing is also keyed on session *number* rather than an exact session-start height match, so a late or undelivered session-start block no longer skips a whole session's clear.

### 🧵 Concurrent proof validation: gas-meter data race fixed

`ValidateSubmittedProofs` fans proof validation across `numCPU` goroutines that all shared a single `sdk.Context`. Every store access runs that context's gas meter, and `gaskv.Store` calls `ConsumeGas` outside any lock — so the parent's `proofIterator.Next()` (seek gas) raced every child's reads (`GetClaim`, signature/closest-path validation, which run outside the coordinator mutex) on `infiniteGasMeter`'s unsynchronized `consumed += amount`.

Each goroutine now gets its own gas meter. Behaviour-neutral: the meter is infinite (EndBlocker) and EndBlocker gas is never reported in `FinalizeBlockResponse`, so nothing consensus-visible was derived from it — state writes were already serialized under the coordinator mutex and are per-claim keyed, so `AppHash` was never at risk. Only the meter is swapped; the MultiStore and EventManager carry over by pointer.

This is a **pre-existing bug** (introduced in #1031, present in v0.1.34 and live on mainnet), not a v0.1.35 regression. It was making `make test_all` / `make test_integration` — which run with `-race` — fail on `x/proof/keeper` and `x/tokenomics/keeper`. Both packages are now race-clean.

### 🚀 Settlement performance

- Aggregation maps keyed by struct instead of `fmt.Sprintf` — removes ~181K string allocations per mainnet settlement block. Sort order is preserved exactly.
- Validator reward data computed once instead of twice per distribution (`calculateAddressRewards` was re-run inside the Largest Remainder Method path).

### 🔐 Supply-chain & docs

- `pocketd-install.sh` now verifies the downloaded tarball against the published `release_checksum` (SHA256) and aborts on mismatch; documents how to review the script before piping it to a shell, and that it installs only the CLI.
- `golang.org/x/crypto` → v0.54.0 (+ `net`, `text`, `sync`, `sys`, `term`, `mod`, `tools`) in both modules.
- Docusaurus `baseUrl` fixed so the docs site loads.

## Consensus-Breaking Changes

1. Settlement budget redistribution + new `overservicing_bonus_multiplier` param (ships as an exact no-op at `m = 1`)
2. Anti-collusion invariant enforced in tokenomics params validation
3. `compute_units_per_relay` resolved at session-start height for claim validation AND settlement; new cupr history state
4. Only the supplier **owner** may cancel an in-progress unstake
5. Dead `GetBlockHash` reads removed from the claim/proof commit-height calculation (**changes gas** on `MsgCreateClaim` / `MsgSubmitProof`)
6. `x/shared` params validation rejects an all-zero claim/proof window offset set (would yield `numPendingSessions = 0` → divide-by-zero panic in settlement → chain halt)
7. `MsgAddService` preserves stored service metadata when the message omits it (previously overwritten with nil by any unrelated update)
8. `Gateway` gains a `metadata` card + new `MsgUpdateGatewayMetadata`; gateway lifecycle events no longer embed the card

## Upgrade Handler (`v0.1.35`)

No KVStore migrations. One deterministic step:

1. **Seed** `overservicing_bonus_multiplier = 1` if unset, then `ValidateBasic()` the resulting params (which also asserts the anti-collusion invariant) before writing them.

> ⚠️ The handler **fails the upgrade** if the live params violate `mint_ratio × supplier < 1`. Verify against the target chain before submitting the upgrade plan:
> ```
> pocketd q tokenomics params --node <rpc>
> ```

## Operator & Governance Notes

- ⚠️ **cupr freeze window.** Do not submit any `MsgAddService` that changes `compute_units_per_relay` between the upgrade height and completion of the RelayMiner fleet rollout. The no-migration design relies on live cupr == historical cupr across that window.
- ⚠️ **Restart the RelayMiner fleet after the upgrade** so miners pick up the session-start cupr stamping.
- ⚠️ **Bulk tokenomics params updates must include every field.** `MsgUpdateParams` (plural) replaces the whole struct — omitting `overservicing_bonus_multiplier` decodes it to `0`, which is coerced to `1`, silently reverting redistribution to OFF. The `bulk_params_main` and `bulk_params_beta` files are backfilled from live chain state and guarded by a test; **`bulk_params_alpha` is NOT verified** (its RPC/gRPC endpoint no longer resolves) — re-derive it from the chain before ever using it.
- **Redistribution stays OFF until governance raises the multiplier.** Enabling it is a treasury decision (absorb the additional settlement, or cut `compute_units_to_tokens_multiplier` for budget neutrality). To lift the cap in practice, set `m ≥ num_suppliers_per_session`, at which point the application budget `B` binds first.
- **Suppliers:** an operator can no longer re-stake to cancel an owner-initiated unstake; that `MsgStakeSupplier` now fails with `PermissionDenied`.
- **Indexers:** new `service/ComputeUnitsPerRelayAtHeight` and `service/ComputeUnitsPerRelayHistory` queries; new `ServiceComputeUnitsPerRelayUpdate` state. `EventApplicationOverserviced` semantics are unchanged at `m = 1`.

---

**Full Changelog**: https://github.com/pokt-network/poktroll/compare/v0.1.34...v0.1.35
