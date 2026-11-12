# Service Supplier Allowlist Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let a service owner restrict which suppliers may serve a service ("permissioned services"): a service with a non-empty supplier allowlist admits only listed suppliers, enforced at stake time, on owner change, and by a batched sweep when the list tightens.

**Architecture:** `x/service` owns the allowlist store, the update message, the queries and the pending-sweep state; a service is permissioned exactly when its list is non-empty (no `Service` field, no flag). `x/supplier` enforces: it rejects stakes, deactivates configs on stake-only owner changes, and runs a per-service EndBlocker sweep that deactivates disallowed configs at the next session start, reusing the v0.1.36 service-cap history machinery.

**Tech Stack:** Go 1.27.2, Cosmos SDK v0.53.8, gogoproto + pulsar (`make proto_regen`), gocuke E2E.

**Spec:** `docs/superpowers/specs/2026-10-14-service-supplier-allowlist-design.md`

## Global Constraints

- Permissioned ⇔ the service's allowlist has at least one entry. No change to `pocket.shared.Service`.
- Every address entering allowlist state is canonical: `sharedtypes.ValidateCanonicalAddress` (x/shared/types/address.go). The list is string-keyed.
- A supplier matches if its `owner_address` or its `operator_address` (its ID) is listed. Revenue share addresses never count.
- `MaxSupplierAllowlistEntriesPerMsg = 100` (x/service/types); `AllowlistSweepBatchSize = 200` (x/supplier/keeper).
- Deactivation is always at the next session start, never immediate, never an unbond. No grace period — except the initial sweep after the first entry, which starts at the next session start after it (Task 7b/7c).
- A sweep request never resets a running sweep: it sets `rerun` (Task 7b). Pending services are served round-robin (Task 7c).
- No params, no `StoreUpgrades`, no upgrade-handler writes for this feature. New prefixes live in the existing `x/service` store.
- EndBlocker code never returns an error and never panics; collect a batch before writing (no write with an open iterator); no map iteration that affects output.
- Proto changes: run `make proto_regen`, then `make go_mockgen` if an interface changed. Tests use `-tags test`.
- `make go_lint` clean before every commit. Commit messages neutral, `[Service] …` / `[Supplier] …` style.
- LOCAL ONLY until release: never push, never open a PR, never tag.

## Review Focus

- Non-canonical (e.g. uppercase) address in `add`/`remove` → rejected in `ValidateBasic`, never stored (Task 1).
- Replacing the only entry in one message (remove A, add B) → the service never reopens, A's suppliers are swept (Task 3).
- Service transferred while its sweep is pending → the new owner controls the list, the old owner is rejected, the pending sweep survives (Task 3).
- Sweep meets a config whose supplier record is missing or undecodable → logged, skipped, no EndBlocker error (Task 7).
- A deactivated, still-unlisted supplier re-stakes the service → rejected at stake; no silent re-entry (Task 5).

---

### Task 1: Protos, message validation, keys and errors (`x/service` types)

**Files:**
- Modify: `proto/pocket/service/tx.proto`, `proto/pocket/service/event.proto`, `proto/pocket/service/query.proto`, `proto/pocket/supplier/event.proto`
- Create: `x/service/types/message_update_service_supplier_allowlist.go`, `x/service/types/key_supplier_allowlist.go`
- Modify: `x/service/types/errors.go`
- Test: `x/service/types/message_update_service_supplier_allowlist_test.go`

**Interfaces:**
- Produces (proto, all in one regen):
  - `MsgUpdateServiceSupplierAllowlist{owner_address=1, service_id=2, repeated string add=3, repeated string remove=4}`, signer `owner_address`; empty `MsgUpdateServiceSupplierAllowlistResponse`; rpc `UpdateServiceSupplierAllowlist`.
  - `EventServiceSupplierAllowlistUpdated{service_id=1, owner_address=2, bool permissioned=3, repeated string added=4, repeated string removed=5}` (`permissioned` = list non-empty after the update).
  - Queries (implemented in Task 4): rpc `SupplierAllowlist(QuerySupplierAllowlistRequest{service_id=1, cosmos.base.query.v1beta1.PageRequest pagination=2}) returns (QuerySupplierAllowlistResponse{bool permissioned=1, repeated string addresses=2, PageResponse pagination=3})`, GET `/pokt-network/poktroll/service/supplier_allowlist/{service_id}`; rpc `SupplierAllowlistEntry(QuerySupplierAllowlistEntryRequest{service_id=1, address=2}) returns (QuerySupplierAllowlistEntryResponse{bool permissioned=1, bool listed=2})`, GET `/pokt-network/poktroll/service/supplier_allowlist/{service_id}/{address}`. The rpc is NOT named `IsSupplierAllowed`: that name is taken by the Keeper method in Task 2 (same receiver).
  - `pocket.supplier.EventSupplierServiceNotAllowed{operator_address=1, owner_address=2, service_id=3, int64 effective_height=4}` (used in Task 6).
- Produces (Go): `const MaxSupplierAllowlistEntriesPerMsg = 100`; `const SupplierAllowlistKeyPrefix = "ServiceSupplierAllowlist/"`; `const SupplierAllowlistSweepKeyPrefix = "ServiceSupplierAllowlistSweep/"`; `func SupplierAllowlistServicePrefix(serviceId string) []byte` (= `serviceId + "/"`); `func SupplierAllowlistKey(serviceId, address string) []byte` (= service prefix + `address`); `ErrServiceInvalidSupplierAllowlist = sdkerrors.Register(ModuleName, 1120, "invalid supplier allowlist update")`.

- [ ] **Step 1: Edit the four protos as listed, run `make proto_regen`, confirm `go build ./...` passes.**

- [ ] **Step 2: Write the failing test `TestMsgUpdateServiceSupplierAllowlist_ValidateBasic`** — table over a valid base msg (canonical owner, `service_id: "svc1"`, one canonical address in `add`). Cases and expected `ErrorIs`:
  - valid → nil; only one `remove` → nil
  - owner `strings.ToUpper(owner)` → `ErrServiceInvalidAddress`
  - `add: [strings.ToUpper(addr)]` → `ErrServiceInvalidSupplierAllowlist`; `remove: ["pokt1invalid"]` → same
  - duplicate within `add`; duplicate within `remove`; same address in `add` and `remove` → `ErrServiceInvalidSupplierAllowlist`
  - 101 entries split 60 add / 41 remove → `ErrServiceInvalidSupplierAllowlist`; exactly 100 → nil
  - empty `service_id` → `ErrServiceMissingID`
  - no `add` and no `remove` → `ErrServiceInvalidSupplierAllowlist` ("no change requested")

- [ ] **Step 3: Run** `go test -tags test ./x/service/types/ -run TestMsgUpdateServiceSupplierAllowlist_ValidateBasic` — expect FAIL (undefined).

- [ ] **Step 4: Implement `(msg *MsgUpdateServiceSupplierAllowlist) ValidateBasic() error`** and the key helpers/consts/error above. Owner check mirrors `MsgTransferService.ValidateBasic`.

- [ ] **Step 5: Run the test** — expect PASS. Run `go test -tags test ./x/service/... ./x/supplier/...` — expect PASS (no behavior change yet).

- [ ] **Step 6: Commit** `[Service] Supplier allowlist protos and message validation`.

### Task 2: Allowlist store, `IsSupplierAllowed`, sweep state (`x/service` keeper)

**Files:**
- Create: `x/service/keeper/supplier_allowlist.go`
- Create: `x/service/types/supplier_allowlist.go` (the `AllowlistSweep` struct)
- Test: `x/service/keeper/supplier_allowlist_test.go`

**Interfaces:**
- Consumes: Task 1 keys.
- Produces:
  - `type AllowlistSweep struct { ServiceId string; Cursor []byte }` (plain Go, `x/service/types`)
  - `func (k Keeper) SetSupplierAllowlistEntry(ctx context.Context, serviceId, address string)`
  - `func (k Keeper) RemoveSupplierAllowlistEntry(ctx context.Context, serviceId, address string)`
  - `func (k Keeper) HasSupplierAllowlistEntry(ctx context.Context, serviceId, address string) bool`
  - `func (k Keeper) IsServicePermissioned(ctx context.Context, serviceId string) bool` — one prefix-iterator `Valid()` check.
  - `func (k Keeper) IsSupplierAllowed(ctx context.Context, serviceId, ownerAddress, operatorAddress string) bool` — true if the list is empty (including an unknown service); else true iff owner or operator is listed.
  - `func (k Keeper) MarkAllowlistSweep(ctx context.Context, serviceId string)` — sets cursor to `[]byte{}` (start or restart).
  - `func (k Keeper) GetPendingAllowlistSweeps(ctx context.Context) []types.AllowlistSweep` — store key order.
  - `func (k Keeper) SetAllowlistSweepCursor(ctx context.Context, serviceId string, cursor []byte)`
  - `func (k Keeper) ClearAllowlistSweep(ctx context.Context, serviceId string)`

- [ ] **Step 1: Write failing tests:**
  - `TestSupplierAllowlistEntries`: set → has true, `IsServicePermissioned` true; remove → false, permissioned false; removing an absent entry is a no-op; entries of `svc1` are invisible to `svc10` (`IsServicePermissioned("svc10")` stays false).
  - `TestIsSupplierAllowed`: table — empty list → true; owner listed → true; operator listed → true; neither listed (another address listed) → false; unknown service → true.
  - `TestAllowlistSweepState`: mark `svcB` then `svcA` → `GetPendingAllowlistSweeps` returns `[svcA, svcB]` with empty cursors; set a cursor, mark again → cursor back to empty; clear → gone.
- [ ] **Step 2: Run** `go test -tags test ./x/service/keeper/ -run 'TestSupplierAllowlistEntries|TestIsSupplierAllowed|TestAllowlistSweepState'` — expect FAIL.
- [ ] **Step 3: Implement** with prefix stores over `SupplierAllowlistKeyPrefix` / `SupplierAllowlistSweepKeyPrefix`, same `runtime.KVStoreAdapter` + `prefix.NewStore` pattern as `x/service/keeper/service.go`. Entry value `[]byte{}`.
- [ ] **Step 4: Run** — expect PASS.
- [ ] **Step 5: Commit** `[Service] Supplier allowlist store`.

### Task 3: `UpdateServiceSupplierAllowlist` message handler

**Files:**
- Create: `x/service/keeper/msg_server_update_service_supplier_allowlist.go`
- Test: `x/service/keeper/msg_server_update_service_supplier_allowlist_test.go`

**Interfaces:**
- Consumes: Task 2 keeper methods.
- Produces: `func (k msgServer) UpdateServiceSupplierAllowlist(ctx context.Context, msg *types.MsgUpdateServiceSupplierAllowlist) (*types.MsgUpdateServiceSupplierAllowlistResponse, error)`.

Rules:
- Errors as gRPC status like `TransferService`: `InvalidArgument` (ValidateBasic), `NotFound` (`ErrServiceNotFound`), `PermissionDenied` (`ErrServiceUnauthorized` when `msg.OwnerAddress != service.OwnerAddress`).
- `before := IsServicePermissioned`; apply removes, then adds, tracking only real changes; `after := IsServicePermissioned`.
- Sweep: `before && !after` → `ClearAllowlistSweep`. `after && (!before || anyRemoved)` → `MarkAllowlistSweep`. Otherwise leave sweep state untouched (adds alone never mark or reset).
- Emit `EventServiceSupplierAllowlistUpdated` once per message, listing only real changes, `permissioned = after`.

- [ ] **Step 1: Write failing tests:**
  - `TestUpdateServiceSupplierAllowlist_Authorization`: non-owner → `ErrServiceUnauthorized`; unknown service → `ErrServiceNotFound`.
  - `TestUpdateServiceSupplierAllowlist_Changes`: add A,B → event `added=[A,B]`, `permissioned=true`; add A again + remove C (absent) → event with empty `added`/`removed`; remove A,B → `permissioned=false`.
  - `TestUpdateServiceSupplierAllowlist_SweepMarking`: table — first add marks; real removal leaving a non-empty list marks; absent removal does not mark; add only leaves an existing cursor `[]byte{1}` untouched; removing the last entry clears a pending sweep; a marking change on a pending service resets cursor `[]byte{1}` to empty.
  - `TestUpdateServiceSupplierAllowlist_ReplaceOnlyEntry`: list `[A]`; one msg `remove A, add B` → still permissioned, list `[B]`, sweep marked (A's suppliers must leave).
  - `TestTransferService_KeepsSupplierAllowlist`: add A + pending sweep, transfer to new owner → entry A and the pending sweep remain; old owner update → `ErrServiceUnauthorized`; new owner update → OK.
- [ ] **Step 2: Run** `go test -tags test ./x/service/keeper/ -run 'UpdateServiceSupplierAllowlist|KeepsSupplierAllowlist'` — expect FAIL.
- [ ] **Step 3: Implement** the handler (no change needed to `TransferService`).
- [ ] **Step 4: Run** — expect PASS.
- [ ] **Step 5: Commit** `[Service] Supplier allowlist update message`.

### Task 4: Queries and CLI

**Files:**
- Create: `x/service/keeper/query_supplier_allowlist.go`
- Modify: `x/service/module/autocli.go`
- Test: `x/service/keeper/query_supplier_allowlist_test.go`

**Interfaces:**
- Consumes: Task 1 query protos, Task 2 store.
- Produces: `func (k Keeper) SupplierAllowlist(ctx, *types.QuerySupplierAllowlistRequest) (*types.QuerySupplierAllowlistResponse, error)` (`query.Paginate` over the service's entry prefix); `func (k Keeper) SupplierAllowlistEntry(ctx, *types.QuerySupplierAllowlistEntryRequest) (*types.QuerySupplierAllowlistEntryResponse, error)`. CLI: `pocketd q service supplier-allowlist <service_id>`, `pocketd q service is-supplier-allowed <service_id> <address>`, `pocketd tx service update-supplier-allowlist <service_id> --add <a,b> --remove <c>` (autocli `PositionalArgs` for `service_id`).

- [ ] **Step 1: Write failing tests** `TestQuerySupplierAllowlist`: 3 entries, `limit 2` → 2 addresses + next key, second page → 1; `permissioned` true, and false for an empty list; unknown service → `codes.NotFound`; nil request → `codes.InvalidArgument`. `TestQuerySupplierAllowlistEntry`: listed → `listed=true`; absent → `false`; non-canonical address → `codes.InvalidArgument`.
- [ ] **Step 2: Run** `go test -tags test ./x/service/keeper/ -run TestQuerySupplierAllowlist` — expect FAIL.
- [ ] **Step 3: Implement** both queries and the three autocli entries.
- [ ] **Step 4: Run** — PASS. `make ignite_pocketd_build` then `pocketd q service supplier-allowlist --help` and `pocketd tx service update-supplier-allowlist --help` print usage.
- [ ] **Step 5: Commit** `[Service] Supplier allowlist queries and CLI`.

### Task 5: Reject disallowed stakes (`x/supplier`)

**Files:**
- Modify: `x/supplier/types/expected_keepers.go` (`ServiceKeeper`), `x/supplier/types/errors.go`, `x/supplier/keeper/msg_server_stake_supplier.go` (after the service-existence loop, ~line 162)
- Test: `x/supplier/keeper/msg_server_stake_supplier_allowlist_test.go`, `tests/integration/migration/morse_claim_supplier_allowlist_test.go`

**Interfaces:**
- Consumes: Task 2 `IsSupplierAllowed`, sweep-state methods.
- Produces: `ServiceKeeper` gains `IsSupplierAllowed`, `GetPendingAllowlistSweeps`, `SetAllowlistSweepCursor`, `ClearAllowlistSweep` (exact Task 2 signatures, `servicetypes.AllowlistSweep`); `ErrSupplierServiceNotAllowed = sdkerrors.Register(ModuleName, 1112, "supplier is not allowed on service")`.

- [ ] **Step 1: Write failing tests** (`keepertest.SupplierKeeper`; its real `ServiceKeeper` shares the store):
  - `TestStakeSupplier_Allowlist`: on `svcId` with another address listed — neither owner nor operator listed → `InvalidArgument` wrapping `ErrSupplierServiceNotAllowed`, message contains `svcId`, `SupplierBalanceMap[owner]` unchanged; only owner listed → OK; only operator listed → OK; empty list → OK; only a rev-share recipient listed → rejected.
  - `TestStakeSupplier_AllowlistRestakeAfterRemoval`: staked while listed, removed (another entry keeps the list non-empty), re-stake listing `svcId` → rejected.
  - `TestMorseClaimSupplier_Allowlist` (integration suite, mirroring the existing Morse supplier-claim tests in `tests/integration/migration/`): claim staking a permissioned service the claimed owner/operator are not listed on → tx fails with `ErrSupplierServiceNotAllowed`.
- [ ] **Step 2: Run** `go test -tags test ./x/supplier/keeper/ -run Allowlist` and `go test -tags test,integration ./tests/integration/migration/ -run Allowlist` — expect FAIL.
- [ ] **Step 3: Implement**: extend the interface and error; add the check (uses `msg.OwnerAddress`, `msg.OperatorAddress`) before any bank call; `make go_mockgen`.
- [ ] **Step 4: Run** both — PASS. `go test -tags test ./x/...` — PASS.
- [ ] **Step 5: Commit** `[Supplier] Enforce service supplier allowlist at stake`.

### Task 6: Deactivation helper and stake-only owner change

**Files:**
- Create: `x/supplier/keeper/service_allowlist.go`
- Modify: `x/supplier/keeper/service_cap.go` (extract shared helpers), `x/supplier/keeper/msg_server_stake_supplier.go`
- Test: `x/supplier/keeper/service_allowlist_test.go`

**Interfaces:**
- Consumes: Task 1 `EventSupplierServiceNotAllowed`, Task 5 interface.
- Produces:
  - `func (k Keeper) nextSessionStartHeight(ctx context.Context) int64` and `func (k Keeper) countedServiceConfigs(ctx context.Context, operatorAddress string, nextSessionStartHeight int64) []*sharedtypes.ServiceConfigUpdate`, both extracted from `EnforceServiceCap` (which then calls them; behavior unchanged).
  - `func (k Keeper) DeactivateSupplierService(ctx context.Context, supplier *sharedtypes.Supplier, serviceId string) (deactivated bool)` — false if unbonding or no counted config for `serviceId`; else `rescheduleServiceConfigDeactivation(…, next session start)` + emit `EventSupplierServiceNotAllowed`. Same caller contract as `EnforceServiceCap` (write the supplier back only with `SetDehydratedSupplier`).
  - `func (k Keeper) enforceSupplierAllowlists(ctx context.Context, supplier *sharedtypes.Supplier) (deactivatedServiceIds []string)` — every counted config whose service disallows the supplier.
  - StakeSupplier: remember the stored owner before `updateSupplier`; if the supplier existed, `len(msg.Services) == 0` and the owner changed, call `enforceSupplierAllowlists` after `SetDehydratedSupplier`.

- [ ] **Step 1: Write failing tests:**
  - `TestDeactivateSupplierService`: counted config → true, `DeactivationHeight == next session start`, one `EventSupplierServiceNotAllowed` with that `effective_height`; second call → false (already deactivating); unbonding supplier → false; service not in the supplier's configs → false.
  - `TestStakeSupplier_OwnerChangeAllowlist`: supplier on `svcId` (permissioned, old owner listed) and `svcId2` (open); stake-only owner change to an unlisted owner → `svcId` deactivates at next session start, `svcId2` untouched, tx succeeds; same but operator listed → nothing deactivates.
  - Existing `EnforceServiceCap` / service-cap sweep tests stay green unmodified.
- [ ] **Step 2: Run** `go test -tags test ./x/supplier/keeper/ -run 'DeactivateSupplierService|OwnerChangeAllowlist|ServiceCap'` — expect FAIL on the new ones.
- [ ] **Step 3: Implement** the extraction, helper, and owner-change call.
- [ ] **Step 4: Run** `go test -tags test ./x/supplier/...` — PASS.
- [ ] **Step 5: Commit** `[Supplier] Deactivate services not allowed after an owner change`.

### Task 7: Per-service allowlist sweep in the EndBlocker

**Files:**
- Create: `x/supplier/keeper/service_allowlist_sweep.go`
- Modify: `x/supplier/module/abci.go` (after `EndBlockerServiceCapSweep`)
- Test: `x/supplier/keeper/service_allowlist_sweep_test.go`

**Interfaces:**
- Consumes: Task 5 interface, Task 6 `DeactivateSupplierService` / `nextSessionStartHeight`, `suppliertypes.IsCountedServiceConfig`, `getServiceConfigUpdatesByServiceStore(ctx, serviceId)`.
- Produces: `const AllowlistSweepBatchSize = 200`; `func (k Keeper) EndBlockerServiceAllowlistSweep(ctx context.Context) (numProcessed int)`.

Algorithm: budget = 200 configs visited per block across all pending services, in `GetPendingAllowlistSweeps` order. Per service: iterate `getServiceConfigUpdatesByServiceStore` from the cursor (exclusive, `append(cursor, 0x00)` as in `EndBlockerServiceCapSweep`), collect up to the remaining budget, close the iterator, then for each config: skip unless `IsCountedServiceConfig`; load the supplier dehydrated (missing → log, skip); skip unbonding; if `!IsSupplierAllowed(serviceId, owner, operator)` → `DeactivateSupplierService`. Set the cursor to the last key visited, or `ClearAllowlistSweep` when exhausted.

- [ ] **Step 1: Write failing tests:**
  - `TestAllowlistSweep_Batches`: 450 counted configs on a permissioned service, none listed → 200, 200, 50 visited over 3 calls; all deactivated; pending cleared after the third.
  - `TestAllowlistSweep_ServicesShareBudget`: services `a` (150 configs) and `b` (150) pending → first call finishes `a` and visits 50 of `b`.
  - `TestAllowlistSweep_CursorReset`: re-mark mid-sweep → next call restarts from the first config.
  - `TestAllowlistSweep_Skips`: unbonding supplier, already-deactivating config, listed supplier, and a config whose supplier record was deleted → none deactivated, no panic, sweep completes.
  - `TestAllowlistSweep_ListEmptiedNoop`: pending service whose list is now empty (handler would have cleared it; plant the state directly) → nothing deactivated, sweep clears.
- [ ] **Step 2: Run** `go test -tags test ./x/supplier/keeper/ -run AllowlistSweep` — expect FAIL.
- [ ] **Step 3: Implement** and wire into `EndBlocker` (log when `numProcessed > 0`).
- [ ] **Step 4: Run** `go test -tags test ./x/supplier/...` — PASS.
- [ ] **Step 5: Commit** `[Supplier] Batched service allowlist sweep`.

### Task 8: Integration and E2E

**Files:**
- Create: `tests/integration/service/supplier_allowlist_test.go`
- Create: `e2e/tests/service_supplier_allowlist.feature`; Modify: `e2e/tests/service_steps_test.go`

- [ ] **Step 1: Write integration tests** (integration app):
  - `TestSupplierAllowlist_SessionMembership`: suppliers S1, S2 staked on a service; owner lists only S2; advance past the next session start (sweep runs) → `GetSession` for the new session contains S2 and not S1; `GetSession` at a height inside the earlier session still contains S1; S1's claim for that earlier session settles (settled event present).
  - `TestSupplierAllowlist_SameBlockUpdates`: S1, S2, S3 staked; in one block the owner sends two update messages (list S2, then list S3) → after the sweep only S1 is deactivated.
- [ ] **Step 2: Run** `go test -tags test,integration ./tests/integration/service/ -run SupplierAllowlist` — PASS once Tasks 1–7 are in (red first if run earlier).
- [ ] **Step 3: Write the Gherkin scenario**: the service owner lists only supplier2 → supplier1 is absent from the next session; the owner adds supplier1 and supplier1 re-stakes → supplier1 is present again. Steps call the Task 4 CLI.
- [ ] **Step 4: Run on a fresh LocalNet** (`make localnet_up`, `make acc_initialize_pubkeys`, `kubectl rollout restart deploy/sage1 deploy/relayminer1`) then `make test_e2e` — all scenarios PASS, including the existing 44.
- [ ] **Step 5: Commit** `[Service] Supplier allowlist integration and E2E coverage`.

### Task 7b: Sweep state with rerun and deferral (`x/service`)

**Files:**
- Create: `proto/pocket/service/supplier_allowlist.proto` (message `AllowlistSweep`)
- Delete: `x/service/types/supplier_allowlist.go` (the plain Go struct is replaced by the generated one)
- Modify: `x/service/keeper/supplier_allowlist.go`, `x/service/keeper/msg_server_update_service_supplier_allowlist.go`
- Test: `x/service/keeper/supplier_allowlist_test.go`, `x/service/keeper/msg_server_update_service_supplier_allowlist_test.go`

**Interfaces:**
- Produces:
  - proto `pocket.service.AllowlistSweep{string service_id = 1; bytes cursor = 2; bool rerun = 3; int64 deferred_from_height = 4;}` — Go `types.AllowlistSweep{ServiceId, Cursor, Rerun, DeferredFromHeight}`; the stored value is the marshaled message; `deferred_from_height == 0` means not deferred.
  - `func (k Keeper) ScheduleInitialAllowlistSweep(ctx context.Context, serviceId string)` — writes `{ServiceId, Cursor: empty, Rerun: false, DeferredFromHeight: ctx block height}`, overwriting any row.
  - `func (k Keeper) RequestAllowlistSweep(ctx context.Context, serviceId string)` — none pending → `{ServiceId}` (start now); pending with `DeferredFromHeight != 0` or empty cursor → no write; pending with non-empty cursor → set `Rerun = true`, cursor kept.
  - `func (k Keeper) SetAllowlistSweep(ctx context.Context, sweep types.AllowlistSweep)` (replaces `SetAllowlistSweepCursor`; stores `[]byte{}`-safe values).
  - `GetPendingAllowlistSweeps` unchanged in name/order; returns the full struct; an undecodable row is logged and skipped (never panics: it runs inside the x/supplier EndBlocker).
  - `MarkAllowlistSweep` and `SetAllowlistSweepCursor` are removed.
- Handler: `!was && is` → `ScheduleInitialAllowlistSweep`; `was && is && len(removed) > 0` → `RequestAllowlistSweep`; `was && !is` → `ClearAllowlistSweep`; adds alone → nothing.

- [ ] **Step 1: Proto + regen** (scratch-clone workaround), delete the hand-written struct, `go build ./...` fails only where `MarkAllowlistSweep` / `SetAllowlistSweepCursor` / the x/supplier interface are used (Task 7c fixes x/supplier; keep it compiling by updating the x/supplier `ServiceKeeper` interface method name and the one call site in `service_allowlist_sweep.go` mechanically — `SetAllowlistSweep(ctx, types.AllowlistSweep{ServiceId: id, Cursor: lastKey})` — behavior changes come in 7c).
- [ ] **Step 2: Failing tests:**
  - `TestAllowlistSweepState` (rewrite): schedule initial at height h → row `{DeferredFromHeight: h}`; request with none pending → `{}` immediate; request on a deferred row → unchanged; request on `{Cursor: [1]}` → `{Cursor: [1], Rerun: true}`; second request → unchanged; request on `{Cursor: empty}` → unchanged; clear → gone; order by service ID.
  - `TestUpdateServiceSupplierAllowlist_SweepMarking` (rewrite rows): first add → deferred row at the block height; real removal leaving non-empty with nothing pending → immediate row; real removal on a started sweep `{Cursor:[1]}` → `Rerun` true, cursor `[1]` kept; absent removal → no change; add only → no change (incl. on `{Cursor:[1]}`); removing the last entry → cleared; replace-only-entry (remove A, add B) → request semantics (was && is && removed).
  - `TestUpdateServiceSupplierAllowlist_RemovalSpam`: started sweep `{Cursor:[1]}`, 10 removal msgs across 10 blocks → row is `{Cursor:[1], Rerun:true}` after every one (never reset).
- [ ] **Step 3: Implement**; run `go test -tags test ./x/service/...` — PASS; `go build ./...` PASS.
- [ ] **Step 4: Commit** `[Service] Allowlist sweep requests never restart a running sweep`.

### Task 7c: Round-robin, deferred start and rerun in the sweep (`x/supplier`)

**Files:**
- Modify: `x/supplier/types/expected_keepers.go`, `x/supplier/types/keys.go`, `x/supplier/keeper/service_allowlist_sweep.go`
- Test: `x/supplier/keeper/service_allowlist_sweep_test.go`

**Interfaces:**
- Consumes: Task 7b `AllowlistSweep{ServiceId, Cursor, Rerun, DeferredFromHeight}`, `SetAllowlistSweep`, `ClearAllowlistSweep`, `GetPendingAllowlistSweeps`.
- Produces: `var AllowlistSweepLastServiceKey = []byte("AllowlistSweep/last_service")` (x/supplier store, value = service ID). `EndBlockerServiceAllowlistSweep` keeps its signature.

Algorithm (replaces Task 7's order and end-of-pass rule):
- Rotate the pending list to start at the first service ID strictly greater than the stored last service, wrapping around.
- Deferred row (`DeferredFromHeight != 0`): due when block height ≥ `GetNextSessionStartHeight(GetParamsAtHeight(DeferredFromHeight), DeferredFromHeight)`; not due → skip with no budget spent and no write; due → process as a fresh pass and write `DeferredFromHeight = 0` with its result.
- Budget, cursor-exclusive iteration, batch-then-write and per-config enforcement unchanged from Task 7.
- End of a service's turn: exhausted && `Rerun` → `SetAllowlistSweep{ServiceId, Cursor: empty, Rerun: false}`; exhausted && !`Rerun` → `ClearAllowlistSweep`; not exhausted → `SetAllowlistSweep{ServiceId, Cursor: lastKey, Rerun: unchanged}`.
- Store the ID of the last service opened this call (not skipped ones) as the last service; no write when nothing was opened.

- [ ] **Step 1: Failing tests** (update Task 7 tests to the new state; keep `Batches`, `Skips`, `ListEmptiedNoop`, `ExactBatchClears` green):
  - `TestAllowlistSweep_RoundRobin`: pending `a`, `b`, `c` with 300 configs each → call 1 visits only `a` (200), call 2 only `b`, call 3 only `c`, call 4 `a` (100) then `b` (100).
  - `TestAllowlistSweep_SpamDoesNotStarve`: `a` (400 configs) re-requested (Rerun set) before every call, `b` (150) pending → `b` is cleared within 2 calls and all its unlisted suppliers are deactivated.
  - `TestAllowlistSweep_RerunRunsOneMorePass`: 50 configs, `Rerun` set mid-pass → the pass finishes, exactly one more full pass runs (configs visited = 100 total), then the row is cleared.
  - `TestAllowlistSweep_DeferredInitial`: row deferred from height h inside a session → calls before the next session start visit 0 and leave the row untouched (other pending services still get the full budget); the first call at/after it processes and the row's `DeferredFromHeight` is 0; deactivation height = the session start after that.
  - `TestAllowlistSweep_BudgetEndsBetweenServices`: `a` exactly 200 configs, `b` pending with cursor `[x]` → `a` cleared, `b` row unchanged, last service = `a`.
  - Replace `TestAllowlistSweep_CursorReset` with the rerun test above.
- [ ] **Step 2: Implement**; `go test -tags test ./x/supplier/...` and `./x/...` — PASS; `make go_lint`.
- [ ] **Step 3: Commit** `[Supplier] Round-robin allowlist sweep with deferred start and reruns`.

### Task 8b: Re-run integration and E2E on the new timing

- [ ] **Step 1:** Update `tests/integration/service/supplier_allowlist_test.go` for the deferred initial sweep (unlisted suppliers leave at the second session start after the first entry; a later removal leaves at the next session start). Add `TestSupplierAllowlist_IncrementalBuild`: on an open service with S1, S2, S3, the owner adds S1 in one block and S2 in a later block of the same session → after the sweep only S3 is deactivated.
- [ ] **Step 2:** Update the Gherkin scenario's timing the same way and run the E2E suite.
- [ ] **Step 3: Commit** `[Service] Supplier allowlist tests for deferred initial sweep`.

### Task 9: Docs, upgrade test, release notes hook

**Files:**
- Modify: the docusaurus page that documents `pocketd tx service transfer-service`, and the supplier staking guide (locate with `grep -rl 'transfer-service\|stake-supplier' docusaurus/docs`)
- Modify: `app/upgrades/v0.1.37.go` doc comment (only if this ships in v0.1.37)

- [ ] **Step 1:** Service-owner section: the first entry permissions the service (already-staked unlisted suppliers leave at the second session start, so finish the list within the current session), removing every entry reopens it, canonical addresses only, owner or operator (the supplier ID) match, effect at the next session start, several messages in one transaction for lists over 100, the list moves with `transfer-service`.
- [ ] **Step 2:** Operator section: a permissioned service rejects a stake unless the owner or operator address is listed; removal takes effect at the next session start; re-stake after being re-added.
- [ ] **Step 3:** If shipping in v0.1.37, add to the handler doc comment: "Permissioned services: `MsgUpdateServiceSupplierAllowlist`; a service with a non-empty supplier allowlist admits only listed suppliers; no state migration."
- [ ] **Step 4:** `make go_lint`; full `go test -tags test,integration ./...` — PASS.
- [ ] **Step 5:** Local upgrade test v0.1.36 → this branch (same procedure as the v0.1.37 upgrade test): no halt at the upgrade height; existing services have empty lists and stay open; a stake on an open service still succeeds; listing one address on a service then rejects an unlisted stake.
- [ ] **Step 6: Commit** `[Docs] Supplier allowlist operator and service-owner docs`.
