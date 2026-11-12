# Service supplier allowlist — design

Date: 2026-10-14
Status: design approved; implementation plan in `docs/superpowers/plans/2026-10-30-service-supplier-allowlist.md`
Target: v0.1.37 if ready in time, otherwise the following release
Depends on: v0.1.36 (shipped; the supplier service cap's history-based deactivation and batched-sweep patterns)

## Goal

Let a service owner restrict which suppliers may serve their service. A service is **permissioned**
when its **supplier allowlist** is non-empty; a supplier may then serve it only if the supplier's owner
address or its operator address (the supplier's ID) is on that list. An empty list means an open service.

Use cases:

- Services whose backend is licensed or private, where only approved operators may serve.
- Quality control: the service owner admits only vetted suppliers.

Non-goals:

- Application allowlists (restricting who may use a service). Any staked application can still use a
  permissioned service. Can be added later as a second list using the same mechanism.
- Governance approval for permissioning. The DAO is not in the permission path.
- Anti-sybil. Not a design driver.
- Reactivating suppliers automatically. A supplier removed from a service re-stakes to serve it again.

## Trust model

The service owner (`Service.owner_address`) controls the allowlist and can permission or reopen the
service at any time, including an existing open service with many suppliers. On mainnet, the owner of a public
service (e.g. eth) can therefore close it with a single transaction. Safeguards:

- Changes take effect only at a session start; current sessions and their claims are unaffected. A
  removal takes effect at the next session start; the first entry (permissioning an open service) takes
  effect for already-staked suppliers at the second session start, giving the owner the rest of the
  current session to finish the list.
- Every change emits an event, so indexers and operators see it immediately.

The service owner also earns the source-owner share of every relay on the service. A permissioned
service lets the owner choose which suppliers earn on it, so owner and suppliers may be the same party.
This is accepted for the use cases above.

## State and messages (`x/service`)

### Permissioned = non-empty list

There is no flag and no change to `pocket.shared.Service`: a service is permissioned exactly when its
allowlist has at least one entry. Every existing service has an empty list, so it stays open and the
upgrade needs no migration. Sessions, which copy `Service`, are untouched.

The list content is the only trigger. This rules out two states a separate flag would allow: a service
switched on with an empty list (closed to every supplier) and a dormant list on an open service. The
cost: a list cannot be prepared before it takes effect (the first entry permissions the service), and
reopening means removing every entry.

### Allowlist storage

A separate store in `x/service`, never embedded in `Service` (sessions copy `Service` and a large list
would bloat every record):

```
ServiceSupplierAllowlist/<service_id>/<address> -> empty value
```

- Entries are canonical (lowercase) bech32 addresses. A supplier is allowed if its `owner_address` **or** its
  `operator_address` is listed. The operator address is the supplier's ID, so listing a supplier by ID is
  listing its operator. Listing an owner covers its whole fleet; listing an operator covers one supplier.
  Revenue share addresses never count.
- The list belongs to the service ID: `MsgTransferService` moves the list with the service.
- Size is unbounded; every entry written or removed is paid for in gas by the service owner.

### Message

```proto
message MsgUpdateServiceSupplierAllowlist {
  option (cosmos.msg.v1.signer) = "owner_address";
  string owner_address = 1;            // must equal Service.owner_address
  string service_id = 2;
  repeated string add = 3;
  repeated string remove = 4;
}
```

The message is judged by the list it leaves behind: removals apply, then additions, and only the
before/after state matters. Replacing the only entry in one message therefore never opens the service,
even briefly. To permission an existing service with more than 100 entries, put several messages in one
transaction: they all apply in the same block, before the sweep runs.

Validation (`ValidateBasic` and the handler):

- Only the current `Service.owner_address` may sign; the service must exist.
- Every address is canonical bech32 (`sharedtypes.ValidateCanonicalAddress`). The list is keyed by the
  address string and compared with the stored owner and operator strings, so a non-canonical entry
  (e.g. all-uppercase) would silently never match.
- No duplicates within `add` or within `remove`; no address in both.
- `len(add) + len(remove)` is at most `MaxSupplierAllowlistEntriesPerMsg` (100, a code constant).
- Adding an existing entry or removing an absent one is a no-op, not an error.
- At least one address in `add` or `remove`.

### Event

```proto
message EventServiceSupplierAllowlistUpdated {
  string service_id = 1;
  string owner_address = 2;
  bool permissioned = 3;               // list non-empty after the update
  repeated string added = 4;           // entries actually added (no-ops excluded)
  repeated string removed = 5;         // entries actually removed (no-ops excluded)
}
```

### Queries

- `pocketd q service supplier-allowlist <service_id>` — paginated list of entries.
- `pocketd q service is-supplier-allowed <service_id> <address>` — whether the address is listed.

## Enforcement

### 1. Staking (`x/supplier` `StakeSupplier`)

Next to the existing `GetService` existence check: for each service in the message, if
the service's allowlist is non-empty and neither the supplier's owner nor its operator address is
listed, reject with `ErrSupplierServiceNotAllowed`, naming the service. Covers `MsgStakeSupplier` and
Morse supplier claims (both stake through `StakeSupplier`). The check runs before any bank transfer or
store write.

### 2. Owner change (`x/supplier` `StakeSupplier`, inline, one supplier)

When a stake message without services changes the supplier's `owner_address`, check the supplier's counted service
configs (active or pending activation, the same predicate the service cap uses) against their services'
allowlists. Configs no longer allowed are deactivated at the next session start. The owner transfer is
never rejected; the supplier keeps only the services the new owner (or its operator) is allowed on.
A stake message that lists services is already covered by the staking check above, which uses the
message's owner and operator.

### 3. Allowlist change: per-service sweep

Trigger — `MsgUpdateServiceSupplierAllowlist` compares the list before and after the message:

- empty → non-empty (the service becomes permissioned): schedule an **initial** sweep, deferred to the
  next session start (see below);
- non-empty → non-empty with at least one entry removed: request a sweep;
- non-empty → empty (the service reopens): clear any pending sweep;
- additions alone never request a sweep and never touch a pending one.

Pending state lives in the `x/service` store, because `x/service` cannot write supplier state:

```
ServiceSupplierAllowlistSweep/<service_id> -> AllowlistSweep{cursor, rerun, deferred_from_height}
```

**Initial sweep is deferred.** The first entry permissions the service at once (new stakes by unlisted
suppliers are rejected immediately), but the sweep over already-staked suppliers starts only at the next
session start after the marking height (`deferred_from_height`). The owner can therefore build the list
over any number of transactions and blocks until then; adds cannot undo a deactivation once scheduled,
so without the deferral an incremental list would kick out suppliers the owner meant to keep. Effect:
already-staked unlisted suppliers leave at the second session start after the first entry.

**Requests never restart a running sweep.** A removal requests a sweep:

- nothing pending → start one now (cursor empty, no deferral);
- pending but not started (deferred, or cursor still empty) → no change: the pass will read the
  latest list;
- pending and started (cursor non-empty) → set `rerun`; the cursor is kept.

When a pass exhausts the service's configs it clears the sweep, or, if `rerun` is set, starts exactly one
more pass from the first config with `rerun` cleared. Every removal is therefore enforced within at most
two passes after it, no matter how many requests follow, and repeated changes cannot hold a sweep at the
start forever. Changes are owner-only and gas-paid; spam can only keep the owner's own service sweeping.

Processing — the `x/supplier` EndBlocker:

- takes pending services **round-robin**: starting with the first service after the last one it opened
  (pointer in the `x/supplier` store), wrapping around, so one busy service cannot starve the others;
- skips, without spending budget, a deferred sweep whose start height has not been reached;
- walks each service's supplier service configs through the existing per-service config index, from the
  cursor;
- processes at most K configs per block across all services (`AllowlistSweepBatchSize`, a code constant,
  starting at 200);
- for each counted config: load the supplier dehydrated, call `IsSupplierAllowed(service, owner,
  operator)`; if not allowed, deactivate that one config at the next session start;
- advances the cursor and, when the service's configs are exhausted, clears it or starts the rerun pass;
- skips unbonding suppliers and configs already deactivating;
- never returns an error and never panics; a failing record is logged and skipped;
- collects each batch before writing, so no write happens while an iterator is open.

Cost is bounded by the suppliers of the swept service, not by the whole network (eth's ~4k suppliers
take about 20 blocks). A sweep may cross a session boundary, in which case some suppliers are
deactivated one session later than others; acceptable for these use cases.

### 4. Deactivation helper

New keeper method in `x/supplier`:

```go
DeactivateSupplierService(ctx, supplier *sharedtypes.Supplier, serviceId string) (deactivated bool)
```

It reloads the supplier's history from the store and sets `DeactivationHeight = nextSessionStartHeight` on
the counted config for that service through `rescheduleServiceConfigDeactivation`, exactly as the shipped
service cap does. The history reload and counting predicate are shared with `EnforceServiceCap` by
factoring them out, not duplicated. Same caller contract as `EnforceServiceCap`: callers write the supplier back only with
`SetDehydratedSupplier`.

Event:

```proto
message EventSupplierServiceNotAllowed {
  string operator_address = 1;
  string owner_address = 2;
  string service_id = 3;
  int64 effective_height = 4;          // next session start
}
```

### 5. Module dependencies

`x/supplier` already depends on `x/service`. Its `ServiceKeeper` interface gains:

```go
IsSupplierAllowed(ctx context.Context, serviceId, ownerAddress, operatorAddress string) bool
GetPendingAllowlistSweeps(ctx context.Context) []AllowlistSweep // service id, cursor, rerun, deferral; store order
SetAllowlistSweep(ctx context.Context, sweep AllowlistSweep)
ClearAllowlistSweep(ctx context.Context, serviceId string)
```

`IsSupplierAllowed` returns true when the service's list is empty. No new dependency direction.

### 6. Unchanged

Session hydration, the slash path, the service cap, settlement and the RelayMiner. A deactivated
service leaves sessions through the service config history, as with the cap and unstaking.

## Consensus safety

- No panics or error returns in the EndBlocker sweep; deterministic store-key iteration; no map
  iteration affecting output.
- Deactivation only at the next session start; past sessions stay reproducible from the history index.
- No params, so no params-history question. The upgrade handler writes nothing for this feature.
- New store prefixes live in the existing `x/service` store: no `StoreUpgrades`.
- No change to `Service`, so session hydration is unchanged byte for byte.
- An upgrade handler that edits allowlists directly must also mark the affected services pending, or
  no sweep runs.

## Testing

Unit (`x/service`):

- Message: only the owner may sign; add/remove idempotent; the first entry permissions the service and
  removing the last reopens it; replacing the only entry in one message never opens it; rejected:
  duplicates, same address in both lists, invalid or non-canonical bech32, over the per-message limit,
  nothing to change; event fields list only real changes.
- Sweep scheduling: empty→non-empty schedules a deferred initial sweep; a removal starts a sweep when
  none is pending, leaves a not-yet-started one alone, and sets `rerun` on a started one (cursor kept);
  cleared on reopening; never on additions alone.
- Several update messages in one transaction all apply before the sweep runs.
- `MsgTransferService` keeps the list.
- Queries, including pagination.

Unit (`x/supplier`):

- Staking: rejected when permissioned and neither address listed; accepted with only the owner listed;
  accepted with only the operator listed; unaffected when the list is empty; Morse claim path rejected the
  same way.
- Owner change: only services the new owner is not allowed on are deactivated at the next session start.
- Sweep: batches across blocks; round-robin across pending services; a deferred initial sweep neither
  runs nor spends budget before its start height; a requested rerun runs exactly one more pass; a
  removal spammed every block never holds the sweep at the start and never starves another service;
  zero-service and unbonding suppliers skipped; a failing record skipped without an EndBlocker error.
- Past sessions: a claim for a session before the deactivation still validates after it.
- `DeactivateSupplierService` and `EnforceServiceCap` share the history logic; both suites stay green.

Integration and E2E:

- Gherkin scenario: the owner lists only supplier2, permissioning the service; supplier1 stops being
  selected from the second session start (deferred initial sweep); adding supplier1 and re-staking
  restores it.
- Settlement for an allowed supplier is unchanged.

Upgrade:

- The v0.1.37 handler needs no code for this feature; its doc comment lists it as consensus-breaking.
- Local upgrade test from v0.1.36.

## Rollout

- No params; nothing changes until a service owner acts.
- Operator docs: a permissioned service rejects a stake unless the supplier's owner or operator address
  is listed; removal takes effect at the next session start; re-stake after being re-added.
- Service-owner docs: how to manage the list; the first entry permissions the service and affects
  existing suppliers at the next session; removing every entry reopens it.
- pocketdex indexes `EventServiceSupplierAllowlistUpdated` and `EventSupplierServiceNotAllowed`.
