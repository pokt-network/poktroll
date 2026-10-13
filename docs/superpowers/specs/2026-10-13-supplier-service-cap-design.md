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
| `min_stake_included_services` | `uint64` | Services covered by `min_stake`. May be 0. |
| `stake_per_additional_service` | `Coin` (upokt) | Stake required per service beyond the included ones. 0 disables the cap. |

Validation: `stake_per_additional_service` is a valid, non-negative upokt coin.
`min_stake_included_services` has no upper bound.

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
- A supplier whose allowed count is 0 is treated exactly like a supplier below
  `min_stake`: it is unbonded with reason `BELOW_MIN_STAKE`. This prevents a supplier
  that is staked but can serve nothing.

With price 0 every path is a no-op and behavior is identical to today.

## Priority order

When a supplier has more active services than it can afford, it keeps the first
`allowed` services in the order of its most recent `MsgStakeSupplier`, and the rest are
deactivated.

This order is already in state: `updateSupplier` appends the new service configs in
message order first, then the older (deactivated) ones. Filtering the history to configs
with no scheduled deactivation preserves the operator's order exactly. No new field is
needed.

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

1. Load the supplier's service config history.
2. Compute `allowed` from its current stake and the live params.
3. Walk the history in stored order, counting configs with no scheduled deactivation.
4. Set `DeactivationHeight = nextSessionStartHeight` on every such config past `allowed`.
5. Persist with `SetAndIndexDehydratedSupplier` so the per-service index used by session
   hydration reflects the change.
6. Emit `EventSupplierServicesCapped` when anything was deactivated.

Deactivation always lands on the next session start, so the current session and its
claims and proofs are unaffected. Session hydration for past heights stays deterministic
because it reads the history index, which records the deactivation height.

### Call site 1: `MsgStakeSupplier`

After the existing `min_stake` check:

- Reject with `ErrSupplierTooManyServices` if `len(msg.Services)` exceeds
  `AllowedServices(msg.Stake)`.
- For a stake-only update (no services in the message), reject if the new stake does
  not cover the supplier's currently active services.

### Call site 2: slashing (`x/tokenomics` `slashSupplierStake`)

- The unbond condition `stake < min_stake` becomes `AllowedServices(stake) == 0`, which
  includes the old condition. The unbonding path itself is unchanged.
- Otherwise call `EnforceServiceCap(…, SLASH)` in place of the plain
  `SetDehydratedSupplier`. When nothing is over the cap this is the same write as today.

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
- Upgrade handler must set both params explicitly. Unset params read as zero values;
  price 0 is the intended launch value, but the handler writes it explicitly rather than
  relying on the default.

## Testing

Unit:

- `AllowedServices`: price 0, `included` 0, stake below min, exact boundaries
  (`min + k·price` ± 1 upokt), large stake without overflow.
- Params: single-field validation and joint combinations; no combination reaches a
  division by zero.
- `MsgStakeSupplier`: reject one over the cap, accept exactly at the cap, reject a
  stake-only decrease below active services, identical behavior with price 0.
- `EnforceServiceCap`: order preserved, next-session deactivation, idempotent, skips
  unbonding suppliers, event fields.
- Past-session determinism: a claim for a session before the deactivation still validates
  after it.
- Sweep: flag set by param update, cursor across blocks, reset on mid-sweep param change,
  flag cleared at end, a failing supplier is skipped without an EndBlocker error.

Integration (`tests/integration`):

- A slash crossing one price step deactivates the last service.
- A slash leaving zero allowed services unbonds with `BELOW_MIN_STAKE`.
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
