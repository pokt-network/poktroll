# Supplier service cap — design

Date: 2026-10-13
Status: design approved in discussion, pending spec review
Target: v0.1.36 (ships disabled), enabled and tightened later by governance

## Goal

Tie the number of services a supplier can serve to its stake. Today a supplier at
`min_stake` can register on any number of services at no extra cost. After this change
each service beyond an included allowance requires additional stake.

Non-goals:

- Per-service or per-tier pricing. The cost lives in one function so tiers can be added
  later without changing callers, but v1 has a single flat price.
- Per-service stake buckets (isolated slashing). Considered and rejected for now: it is a
  different data model, needs a stake-splitting state migration, and slashes today are
  small relative to stake.
- Ongoing per-service rent. A likely follow-up; the pooled cap is designed so that rent
  draining stake automatically sheds services through this mechanism.
- Changing `min_stake`.

## Parameters

Two new `x/supplier` params, set through `MsgUpdateParam` / `MsgUpdateParams` like
`min_stake`:

| Param | Type | Meaning |
|---|---|---|
| `min_stake_included_services` | `uint64` | Services covered by `min_stake`. At least 1. Default 1. |
| `stake_per_additional_service` | `Coin` (upokt) | Stake required per service beyond the included ones. 0 disables the cap. |

Validation: `stake_per_additional_service` is a valid, non-negative upokt coin.
`min_stake_included_services` must be at least 1 and has no upper bound.

Why at least 1: a staked supplier exists to serve. With 0, a supplier between `min_stake`
and `min_stake + price` would be staked but allowed no services. Any pricing expressed
with 0 included services is expressed identically with 1 by raising `min_stake` by one
price step (`min 5k, included 0, price 5k` requires the same stake for every service
count as `min 10k, included 1, price 5k`). Allowing 0 later would be a consensus change
(message validation) that also has to add handling for the zero-services state.

`stake_per_additional_service` is a pointer `Coin`, like `min_stake`, and `GetParams` can
return it nil. Every reader treats a nil price as 0 (cap disabled), the same way the
existing code falls back to `DefaultMinStake` for a nil `min_stake`.

These params have no history store and are always read live. That is correct here: the
cap only decides which services are active from the *next* session start, i.e. it
schedules a future height and never prices past work.

## Allowed services

One pure function in `x/supplier/types`:

```
AllowedServices(stake, params) -> (allowed uint64, unlimited bool)

if price == 0:            unlimited
if stake < min_stake:     0
otherwise:                included + floor((stake - min_stake) / price)
```

- Price 0 returns before any division, so no parameter combination divides by zero.
- Required stake for `n` services: `min_stake + max(0, n - included) * price`.
- Because `included >= 1`, the allowed count is 0 exactly when stake is below
  `min_stake`. The existing below-minimum unbonding covers it; this design adds no new
  unbonding rule.

With price 0 every path is a no-op and behavior is identical to today.

## Priority order

When a supplier has more active services than it can afford, it keeps the first
`allowed` services in the order of its most recent `MsgStakeSupplier`, and the rest are
deactivated.

The service config history does not carry this order: it is reloaded from the index
keyed `operator | serviceId | activationHeight`, so a reload returns configs sorted by
service ID. The order is therefore stored explicitly:

- New field on `Supplier`: `repeated string service_priority`, the service IDs of the
  most recent `MsgStakeSupplier` in message order.
- `updateSupplier` (and the new-supplier path in `StakeSupplier`) writes it from
  `msg.Services`. A stake-only update leaves it unchanged.
- `SetDehydratedSupplier` clears only `Services` and `ServiceConfigHistory`, so the field
  persists on the dehydrated record.
- `EnforceServiceCap` ranks active configs by their position in `service_priority`.
  Services missing from the list (every supplier staked before this release, until it
  restakes) rank after listed ones, in service ID order. This fallback is deterministic
  and is shown in the pre-tightening impact list. The upgrade handler cannot backfill the
  field because the original message order is not recoverable from state.

Rejected alternative: "newest activation first". Any restake that lists services gives
every listed service a fresh activation height, so activation height carries no tenure
information, and pruned history cannot recover it.

Operators control priority by ordering their stake config. Re-staking with a subset is an
explicit choice of which services to keep.

## Enforcement

### `EnforceServiceCap` (x/supplier keeper)

```
EnforceServiceCap(ctx, supplier, reason) -> deactivatedServiceIds
```

1. Compute `allowed` from the supplier's current stake and the live params. If the cap
   is disabled, or the supplier is unbonding, return. A supplier below `min_stake` is
   left to the existing unbonding path.
2. Load the supplier's service config history from the store.
3. Take the configs with no scheduled deactivation (`DeactivationHeight == 0`: active now
   or pending activation) and rank them by `service_priority`.
4. Set `DeactivationHeight = nextSessionStartHeight` on every config ranked past
   `allowed`.
5. Persist with `SetAndIndexDehydratedSupplier` so the per-service index used by session
   hydration reflects the change.
6. Emit `EventSupplierServicesCapped` when anything was deactivated.

Callers pass the supplier pointer they already hold. In settlement this must be the
`settlementContext` cached pointer: `FlushAllActorsToStore` later rewrites the dehydrated
record from that cache. The flush does not touch the service config indexes, so the
deactivations written here survive it.

Deactivation always lands on the next session start, so the current session and its
claims and proofs are unaffected. Session hydration for past heights stays deterministic
because it reads the history index, which records the deactivation height.

### Call site 1: `StakeSupplier`

The check goes in the keeper's `StakeSupplier`, next to the existing `min_stake` check,
not in the message handler. `MsgStakeSupplier` and Morse supplier claims
(`x/migration` `msg_server_claim_morse_supplier.go`) both stake through `StakeSupplier`,
so neither bypasses the cap. Genesis import and existing migrations do not register
services through user input and are not call sites.

- Reject with `ErrSupplierTooManyServices` if `len(msg.Services)` exceeds
  `AllowedServices(msg.Stake)`.
- For a stake-only update (no services in the message), reject if the new stake does
  not cover the configs with no scheduled deactivation (the same set `EnforceServiceCap`
  counts: active now or pending activation).

### Call site 2: slashing (`x/tokenomics` `slashSupplierStake`)

- The unbond condition `stake < min_stake` is unchanged.
- Otherwise call `EnforceServiceCap(…, SLASH)` on the cached supplier in place of the
  plain `SetDehydratedSupplier`. When nothing is over the cap this is the same write as
  today.
- Today's `else` branch writes only the dehydrated record because "the service config will
  remain unchanged". That assumption no longer holds. A second slash on the same supplier
  in the same settlement reloads history from the store and sees the deactivations already
  scheduled.

### Call site 3: param-change sweep (`x/supplier` EndBlocker)

Only governance changes to `min_stake`, `min_stake_included_services` or
`stake_per_additional_service` can make an untouched supplier exceed its cap. Slashes and
stakes are handled inline, so a periodic full scan is unnecessary.

- A successful update to any of those three params sets a pending-sweep flag in state and
  resets a cursor.
- While the flag is set, the EndBlocker processes up to `K` suppliers per block in key
  order from the cursor, calling `EnforceServiceCap(…, PARAMS)` on each non-unbonding
  supplier, then advances the cursor.
- When the cursor reaches the end of the supplier store, the flag is cleared.
- A param change during a sweep resets the cursor so the sweep restarts with the new
  values.
- When the cap is disabled (price 0), the sweep does nothing beyond clearing the flag.
- Errors for an individual supplier are logged and skipped. The EndBlocker never returns
  an error from this loop, since that halts the chain.

`K` is a code constant, not a param. It ships conservative and is tuned from mainnet
measurement (see Rollout).

Measured cost (in-memory test store, 4,400 suppliers × 10 services): reading a supplier
without history costs ~1.5k gas; with history ~22k gas. A batch of 200 suppliers is
~4.5M gas of reads. Wall time in the test store is dominated by uncommitted-cache
iteration and is not representative of mainnet; real time must be measured on a mainnet
database copy.

New state: one flag key and one cursor key in the `x/supplier` store.

### Event

```
EventSupplierServicesCapped {
  operator_address
  owner_address
  stake
  allowed_services
  deactivated_service_ids
  reason              // SLASH | PARAMS
  effective_height    // next session start
}
```

## Consensus safety

- No panics on any path. No error returns from the EndBlocker sweep.
- Deterministic iteration (store key order). No map iteration affects output.
- Live params only, which is correct for scheduling the next session.
- Session hydration is untouched; it continues to read the service config history index.
- Only `MsgUpdateParam(s)` sets the pending-sweep flag. An upgrade handler that changes
  `min_stake`, `min_stake_included_services` or `stake_per_additional_service` with
  `SetParams` must also set the flag, or no sweep runs. Same class of trap as the shared
  params history note in CLAUDE.md.
- A nil `stake_per_additional_service` is treated as 0 on every path.
- Upgrade handler must set both params explicitly and to values that pass `Validate()`
  (`min_stake_included_services = 1`, `stake_per_additional_service = 0`). A handler
  writing params that fail validation halts the chain. Unset params read as zero values;
  price 0 is the intended launch value, but the handler writes it explicitly rather than
  relying on the default.

## Testing

Unit:

- `AllowedServices`: price 0, `included` 0, stake below min, exact boundaries
  (`min + k·price` ± 1 upokt), large stake without overflow.
- Params: single-field validation (including `min_stake_included_services = 0` rejected)
  and joint combinations; no combination reaches a division by zero.
- `MsgStakeSupplier`: reject one over the cap, accept exactly at the cap, reject a
  stake-only decrease below active services, identical behavior with price 0.
- `EnforceServiceCap`: order preserved through the store (stake, reload the supplier
  from the store, assert the kept services follow `service_priority`, not the slice
  `updateSupplier` returned); fallback order for suppliers with an empty
  `service_priority`; next-session deactivation; idempotent; skips unbonding suppliers;
  never unbonds; event fields.
- Nil `stake_per_additional_service` behaves as a disabled cap.
- Morse supplier claim with more services than the stake covers is rejected.
- Past-session determinism: a claim for a session before the deactivation still validates
  after it.
- Sweep: flag set by param update, cursor across blocks, reset on mid-sweep param change,
  flag cleared at end, a failing supplier is skipped without an EndBlocker error.

Integration (`tests/integration`):

- A slash crossing one price step deactivates the last service.
- A slash below `min_stake` still unbonds with `BELOW_MIN_STAKE`, unchanged.
- Two `PROOF_MISSING` claims for the same supplier in one settlement block: the second
  slash sees the reduced service set, and the end-of-settlement flush does not undo the
  deactivations.
- Settlement payouts for suppliers under the cap are identical with the cap on and off.

E2E:

- Gherkin scenario: a param update sets a price; an over-cap supplier loses its last
  service at the next session; a top-up restake restores it.

Upgrade:

- Local upgrade from v0.1.35: params present, price 0, no behavior change, `gas_used`
  differs only by the added param reads.

## Rollout

1. v0.1.36 ships the code with `stake_per_additional_service = 0` (disabled).
2. Before the first enablement:
   - Measure one sweep batch on a mainnet database copy and set `K` so a batch stays a
     small fraction of block time.
   - Run an impact script that lists, for the proposed values, each supplier's allowed
     count, current count, and the services it would lose.
3. First enablement: a real price with `min_stake_included_services` set to the current
   network maximum services per supplier, so nobody is over the cap.
4. Monthly tightening by param transactions, each announced in advance with its impact
   list. Operators top up or reorder their stake config before it lands.
5. Operator-facing: document that stake config order is service priority, add a
   `pocketd` warning when listed services exceed what the stake covers, and index
   `EventSupplierServicesCapped` in pocketdex.

The HA RelayMiner needs no change: deactivated services leave sessions through the
existing service config history mechanism.
