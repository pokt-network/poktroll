# Plan: Pin `compute_units_per_relay` to session-start (chain + miner)

**Status:** Planned — not yet implemented.
**Author:** Otto V (Head of Protocol)
**Date:** 2026-06-23

## Problem

Changing a service's `compute_units_per_relay` (cupr) while sessions for that
service are open/unclaimed causes those claims to be **rejected at claim creation**
with `ErrProofComputeUnitsMismatch` (proof code **1126**), forfeiting the reward.
The work is unrecoverable (the SMST is append-only).

Observed 2026-06-22: a batch of services (incl. `seda`) changed cupr; the operator
`kalorious` (the last one running the stock `pocketd` relayminer) forfeited the
affected in-flight claims. HA-binary operators hit the same class of bug earlier
(`pocket-relay-miner`).

## Root cause — a chain/miner semantics mismatch

The two sides disagree on **which height's cupr** applies to a session:

- **Chain checks LIVE cupr at claim-creation height.**
  `x/proof/keeper/msg_server_create_claim.go:95,98`:
  `numExpectedComputeUnits = numRelays * cupr`; mismatch → 1126.
  cupr comes from `x/proof/keeper/service.go:18` → `serviceKeeper.GetService(ctx)`
  = current param at the claim-creation block.

- **Miner bakes cupr per-relay at MINE-time** into the append-only SMST.
  `pkg/relayer/session/session.go:623`: `smst.Update(relay.Hash, relay.Bytes, cupr)`,
  weight from `pkg/relayer/session/service.go:17`.
  The weight is frozen when each relay arrives; the SMST can't be reweighted.

Therefore, for any session whose relays were mined under the old cupr but claimed
after a cupr change: `treeSum(old cupr) != numRelays * cupr(new)` → 1126 → forfeit.

### Stock cache behaviour (corrected 2026-06-25)

`serviceQueryClient.GetService` uses a KeyValueCache constructed with
`WithTTL(math.MaxInt64)` (`pkg/deps/config/suppliers.go:533`) **but also**
`WithSessionCountCacheClearFn(defaultSessionCountForCacheClearing)` with interval
**= 1** (`pkg/relayer/cmd/deps.go:26,111`), which clears the whole service cache **at
every session start** (`pkg/client/query/cache/options.go:44`). So the `MaxInt64`
TTL is only a time-fallback — the per-session clear dominates, and cupr is
re-fetched fresh each session (~64 min on mainnet N=60). There is **no
process-lifetime freeze** in stock (an earlier draft of this doc claimed there was
— that was wrong; that freeze was the HA fork's separate `sync.Map` provider).

Net: stock effectively pins cupr to ~session-start already. The forfeit is **not**
staleness — the miner bakes session-start cupr (correct) while the chain checks
**live** cupr at claim creation, so any cupr change between session-start and claim
mismatches. A blunt time-based TTL on the stock cache is therefore redundant and
does not fix forfeits; the chain session-start pin (below) does.

### The asymmetry to fix

Relay-mining **difficulty is already pinned to session-start** on both sides
(`x/proof/keeper/msg_server_submit_proof.go:128-131` reads
`GetParamsAtHeight(sessionStartHeight)` and
`GetRelayMiningDifficultyAtHeight(serviceId, sessionStartHeight)`).
cupr is the lone field still read live. The fix is to make cupr behave like
difficulty.

## Decisions made (and rejected)

- **Refresh the cache faster — REJECTED.** Mid-session refresh produces mixed-weight
  trees (some relays old cupr, some new) → non-integer sum → fails regardless of which
  cupr the chain uses. The long TTL is protective, not the bug.
- **Re-stamp the whole tree at claim-build with current cupr — VALID but DROPPED.**
  It works (proof validation is weight-agnostic — see below — and the chain only checks
  the aggregate once at claim creation) and is NOT an abuse vector (the supplier has zero
  degrees of freedom: `numRelays` is difficulty- + signature-gated, and cupr is forced
  to `numRelays * cupr_live`). But it conforms to the *live*-cupr semantics, which is the
  thing we want to change. Superseded by the session-start pin.
- **Pin cupr to session-start on chain + miner — CHOSEN.** Durable, principled, matches
  the existing difficulty model. cupr changes then apply only to sessions starting after
  the change; in-flight sessions keep their start rate.

### Supporting facts (verified)

- Proof validation never checks per-leaf cupr/weight — only relay difficulty
  (`validateRelayDifficulty`), signatures, closest-path, and session header
  (`x/proof/keeper/proof_validation.go`).
- cupr is checked exactly once, as an aggregate, at claim creation
  (`msg_server_submit_proof.go:118` DEV_NOTE).
- Settlement uses the claim's baked `numClaimComputeUnits` (from the root) — it does
  not re-query cupr. So a landed claim settles consistently; no settlement-time cupr
  re-check.
- In `smst.Update(hash, bytes, weight)` the weight is a separate arg from the leaf
  identity/value; changing it alters the tree sum, not leaf membership or the proof.
- `tests/integration/.../unhaltable_test.go:20` lists "compute units per relay is
  updated mid-session" as a known hazard — but the test is an unimplemented stub.

## Target end state

Single source of truth: **cupr-at-session-start**. The chain checks it; the miner
stamps it. Consistent by construction. A cupr change applies only to sessions that
start after it.

## Chain fix (consensus-breaking) — mirror difficulty-at-height

Template to copy: `x/service/keeper/relay_mining_difficulty.go` +
`x/service/keeper/query_relay_mining_difficulty.go` +
`x/service/keeper/update_relay_mining_difficulty.go`.

1. **Storage + accessors.** Add `SetServiceComputeUnitsPerRelayAtHeight` /
   `GetServiceComputeUnitsPerRelayAtHeight` to `x/service/keeper`, snapshotting the
   **previous** cupr at the change height, with **fallback to current** when no
   snapshot exists (exactly as difficulty does). Snapshot-on-change is wired into the
   cupr update path at `x/service/keeper/msg_server_add_service.go:63` (AddService
   doubles as update).
2. **Query endpoint.** Add `ComputeUnitsPerRelayAtHeight(serviceId, blockHeight)` to
   the service module's `Query` service (proto + autocli), so the miner can fetch
   session-start cupr. (See CLAUDE.md "Adding Query Endpoints".)
3. **Use session-start cupr in the claim check.** Change
   `x/proof/keeper/service.go` / `msg_server_create_claim.go:87` to call
   `GetServiceComputeUnitsPerRelayAtHeight(ctx, serviceId, sessionStartHeight)`
   instead of live `GetService(ctx)`. `sessionStartHeight` is available from the
   claim's session header.
4. **Gate the switch in an upgrade handler** at height `H` (`app/upgrades/`). The
   storage + query endpoint can ship un-gated (additive, non-breaking); only the
   claim-check switch from live → session-start is consensus-breaking and must
   activate at `H`.

Genesis export/import: include any new cupr-history KV state (see CLAUDE.md
"When adding new KV state").

## Miner fix (binary) — stamp session-start cupr

1. Replace the cached-live per-relay stamp in
   `pkg/relayer/session/service.go:17` / `session.go:615-623` with a **session-start**
   cupr: query `ComputeUnitsPerRelayAtHeight(serviceId, sessionStartHeight)` once per
   session, pin it, and weight all relays in that session uniformly.
   - `sessionStartHeight` is on the relay request's session header.
   - This removes the dependency on the process-global service cache for cupr.
2. **(Defensive)** Classify `ErrProofComputeUnitsMismatch` (1126) as a permanent
   error in the submit/rebroadcast path → mark the session failed, stop retrying,
   surface it. Prevents wasted gas sims / log spam on any residual mismatch.

## Sequencing — the cupr-change freeze is the bridge

Both sides derive cupr from session-start, so any old/new mix is safe **as long as
cupr does not change during the rollout** (then live == session-start == cached and
all combinations match). Transition matrix:

| miner \ chain | old chain (live) | new chain (session-start) |
|---|---|---|
| old miner (cached-live) | works if cupr frozen | works if cupr frozen |
| new miner (session-start) | works if cupr frozen | works always |

Steps:

1. **Freeze all cupr changes now.** Let kalorious's already-doomed `seda` (and other
   changed-cupr) in-flight sessions expire — unrecoverable.
2. **Release binary `vN`:** cupr-at-height storage + query + snapshot-on-change record
   immediately; claim check stays live pre-`H`, flips to session-start at `H` via the
   upgrade handler.
3. **Validators upgrade to `vN`**, schedule upgrade height `H`.
4. **Miners upgrade to the `vN` relayer** (session-start stamping) — any time after
   nodes serve the new query, before `H`.
5. **At `H`:** claim check flips to session-start. Both sides now session-start.
6. **Confirm the miner fleet is on `vN`**, then **unfreeze** cupr changes.

## Testing

- Chain: unit tests for `Get/SetServiceComputeUnitsPerRelayAtHeight` (snapshot-on-change,
  fallback-to-current) mirroring the difficulty tests; integration test —
  cupr changed mid-session, claim built with session-start cupr settles cleanly
  (this is the scenario stubbed at `unhaltable_test.go:20`).
- Miner: session-start cupr is stamped uniformly even when live cupr differs;
  no mixed-weight trees across a simulated cupr change.
- E2E on LocalNet/betanet: change a service's cupr mid-session, confirm in-flight
  claims (built post-fix) settle instead of forfeiting.

## Risks / notes

- Consensus-breaking — requires coordinated upgrade. No KV migration beyond the new
  cupr-history store (snapshot-on-change starts recording from `vN`; pre-`vN` history
  falls back to current, which is correct because cupr is frozen through the window).
- Keep the rebroadcast fix (`fix/relayminer-proof-rebroadcast`) as a separate,
  independent change — it is cupr-safe (inclusion-keyed, no 1126 retry-spam) and ships
  on its own track.
- Out of scope: a settlement-time cupr re-check (not needed — settlement uses the
  claim's baked compute units).

## Reconciliation with HA miner fix `#14` (`82d91aa`, 2026-06)

Jorge shipped an HA cupr fix in a different direction than this plan — **review before implementing the chain change.**

- **`#14` approach:** replace the process-lifetime `sync.Map` cupr cache with a
  TTL + pub/sub-refreshed service cache (`relayer/service_compute_units_provider.go`),
  floor-to-1 on error/zero. This is *live*-cupr tracking — pick up a change within
  the refresh/invalidation window.
- **What it fixes:** the *freeze* bug (cupr frozen for process life → every session
  after a change forfeits until restart). Now sessions that **start** after a change
  use the new value. Blast radius: "every session until restart" → "only a session
  active mid-flight at the change instant."
- **What it does NOT fix:** a session **open during** the change still mixes old/new
  weights (append-only SMST + live chain check) → 1126. This plan's session-start
  pin is what eliminates that remaining case.
- **Divergence / decision needed:** `#14` tracks *live* cupr; this plan pins
  *session-start*. They don't conflict today, but if the chain check switches to
  session-start (step 3 above), the HA miner must switch from live-tracking to
  session-start stamping or it will mismatch. Agree the end-state with Jorge first:
  (a) session-start everywhere (durable — this plan), or (b) live-refresh everywhere
  + operational freeze (mitigation only, leaves transitional forfeits).
- **Stock parity note (corrected):** `#14` fixed a *freeze* bug specific to the HA
  fork's `sync.Map` cupr provider. Stock poktroll does NOT have that freeze — its
  service cache clears every session start (`deps.go:111`, interval=1), so cupr
  already refreshes per session. No stock TTL/refresh change is needed or helpful;
  it would be redundant and would not fix forfeits.
- **Proof reconciler note:** `#14` also hardened the HA proof reconciler (count a
  rebroadcast attempt even on failure; log at Debug). The stock rebroadcast branch
  (`fix/relayminer-proof-rebroadcast`) already does both — no change needed.

## Workstreams / branches

1. `fix/relayminer-proof-rebroadcast` — DONE, committed (`8090cbbe6`), independent.
2. Chain: cupr-at-height (storage + query + claim-check switch + upgrade handler) — **IMPLEMENTED** on branch `feat/cupr-session-start-pin` (uncommitted, pending Otto review).
3. Stock miner: session-start cupr stamp — **IMPLEMENTED** on the SAME branch `feat/cupr-session-start-pin`. The stock RelayMiner now weights every relay by `GetServiceComputeUnitsPerRelayAtHeight(serviceId, sessionStartHeight)` (new immutable, never-cleared `KeyValueCache[uint64]`) instead of the live per-session-cleared service cache — so it is robust to the node-stall / exact-equality cache-clear mixed-tree bug, not just the plain live-vs-session-start asymmetry.
   - Files: `pkg/client/interface.go`, `pkg/client/query/servicequerier.go`, `pkg/relayer/cmd/deps.go`, `pkg/relayer/session/service.go`.
   - 1126-permanent classification: still OPTIONAL / deferred (not needed now that the miner and chain agree on session-start cupr).
4. HA miner: realign to session-start stamping — AFTER this lands (separate repo). See the reconciliation section above.
