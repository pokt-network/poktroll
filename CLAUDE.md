# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Poktroll is a Cosmos SDK-based blockchain implementing Pocket Network's Shannon upgrade - a decentralized API layer for Web3. Built with Go 1.25.8, Cosmos SDK v0.53.7, and CometBFT consensus (via the `pokt-network/cometbft` fork, pinned by a `replace` directive in `go.mod` - upstream `cometbft/cometbft` is never used directly).

The binary is `pocketd`.

## Development Commands

The root `Makefile` is thin - nearly every target lives in a topic file under `makefiles/*.mk`
(`tests.mk`, `localnet.mk`, `params.mk`, `release.mk`, `claims.mk`, `migrate.mk`, ...).
Run `make help` to list targets with descriptions, and grep `makefiles/` rather than the root
`Makefile` when looking for a target.

### Core Development

```bash
make go_develop                 # Generate protos and mocks (run after proto changes)
make go_develop_and_test       # Generate + run all tests
make ignite_pocketd_build      # Build pocketd binary to GOPATH/bin
make proto_regen               # Regenerate protobuf artifacts
```

### Testing

```bash
make test_all                  # Run all tests
make test_integration          # In-memory integration "unit" tests
make test_e2e                  # All E2E Gherkin scenarios (needs LocalNet)
make go_lint                   # Run linters (always run before commits)
```

E2E suites are runnable individually, which is much faster than `test_e2e` when iterating on one
module: `test_e2e_relay`, `test_e2e_app`, `test_e2e_supplier`, `test_e2e_gateway`,
`test_e2e_session`, `test_e2e_tokenomics`, `test_e2e_params`. See `makefiles/tests.mk` for the
full list, plus the `test_load_relays_stress_*` load-test targets.

### LocalNet Operations

```bash
make localnet_up               # Start local development network
make localnet_down             # Stop local network
make localnet_reset            # Reset and restart network
make acc_balance_query ACC=<addr>  # Query account balance
```

## Architecture

### Core Modules (`/x/`)

- **application** - App staking and delegation for API access
- **supplier** - Service provider (RelayMiner) management
- **gateway** - Quality-of-service layer for enterprise usage
- **service** - API service registry and relay mining difficulty
- **session** - Time-bounded interaction windows between apps/suppliers
- **proof** - Cryptographic verification of API usage for settlements
- **tokenomics** - Economic incentives, penalties, and token distribution
- **shared** - Cross-module utilities and constants
- **migration** - State migration logic for chain upgrades

### Key Components

- **`/pkg/relayer/`** - RelayMiner implementation for API proxying
- **`/pkg/client/`** - Blockchain client libraries and query helpers
- **`/pkg/crypto/`** - Ring signatures and cryptographic utilities
- **`/pkg/observable/`** - Reactive programming patterns using channels
- **`/pkg/polylog/`** - Structured logging framework (use instead of standard log)
- **`/pkg/cache/`** - `KeyValueCache` / `HistoricalKeyValueCache` interfaces. Offchain use only
  (RelayMiner, clients) - never on a Keeper, see Consensus Safety below
- **`/pkg/encoding/`** - Deterministic conversions, notably `Float64ToRat` for float params
- **`/pkg/cards/`** - Validation for Pocket "cards": self-describing JSON metadata documents.
  The chain enforces size only and never parses them; the canonical schemas live here
- **`/pkg/faucet/`** - HTTP funding server that broadcasts bank sends (`/{denom}/{address}`)
- **`/pkg/sync2/`** - Concurrency `Limiter` (vendored from Storj)
- **`/pkg/pokterrors/`** - `Split` for unwrapping `errors.Join` trees

### Development Patterns

- **Protocol-first development** - Always update `.proto` files before implementation
- **Keeper pattern** - State management through module keepers with proper gas metering
- **Event-driven architecture** - Emit typed events for cross-module communication
- **Observable patterns** - Use `pkg/observable` for reactive data flows
- **Ring signatures** - Privacy-preserving authentication in `pkg/crypto/rings`

### Consensus Safety (Critical)

These rules prevent AppHash mismatches and chain halts. Violating them will cause non-determinism across validators:

- **No in-memory caches on Keeper structs** - Keepers are shared across nodes; caches diverge and cause AppHash mismatch. Use block-scoped local variables instead.
- **Sort map keys before iteration** - Go map iteration order is non-deterministic. Always sort keys when iterating maps in state-changing code (BeginBlocker, EndBlocker, message handlers).
- **Use `ctx.BlockTime()`, never `time.Now()`** - `time.Now()` differs across validators and will cause consensus failure.
- **Use `encoding.Float64ToRat` for float64 params** - Float64 arithmetic is non-deterministic across platforms. Convert to `big.Rat` for deterministic computation.
- **`MustUnmarshal(nil)` does not panic with gogoproto** - It silently returns a zero-value struct with nil message fields. Add nil checks when reading from secondary indexes that may contain orphaned entries.
- **Params-at-height is available for `shared` only; settlement uses it selectively** - `x/shared` maintains a params history (`recordParamsHistory`, `GetParamsAtHeight`, `IterateParamsHistoryReverse`); `x/session` exposes `GetParamsAtHeight` over it. No other module has a history store, so their `GetParams(ctx)` is always the *live* value. Settlement is a mix, by design and by omission:
  - **At-height:** the expiring-claim scan walks params history per-epoch (`candidateSessionEndHeightsForLiveParams`) instead of solving against live offsets — `GetExpiringClaimsIterator` is deprecated and no longer on the settlement path. The budget divisor reads `GetParamsAtHeight(sessionEndBlockHeight)`; compute-units-per-relay resolves via `GetServiceComputeUnitsPerRelayAtHeight(sessionStartHeight)`; and **claim pricing** (`compute_units_to_tokens_multiplier`, `compute_unit_cost_granularity`) resolves via `settlementContext.GetSharedParamsAtHeight(sessionStartHeight)`, which is what makes settlement agree with `x/proof`.
  - **Still live, deliberately:** `settlementContext.GetSharedParams()` is the live snapshot and MUST stay live for its remaining callers — they project *future* heights from the *current* block (supplier unbonding, application unbonding, session end height), which need the grid in effect now. Also live: tokenomics params (mint ratios, distribution), `TargetNumRelays`, and proof/supplier/application params, none of which have a history store to read from.

  When adding a param that settlement consumes, decide explicitly which side it belongs on: **pricing → session start; scheduling a future height → live**. Anything on the live side changes payouts for already-claimed, not-yet-settled work the moment governance updates it.
- **An upgrade handler that writes shared params with `SetParams` alone is invisible to params history** - Only `msgServer.recordParamsHistory` (the `MsgUpdateParam(s)` path) writes history; a handler calling `SharedKeeper.SetParams` moves live and nothing else. Since every at-height consumer (`x/proof`, settlement pricing, the budget divisor, the RelayMiner) resolves from history, such a write **silently does not take effect for those consumers** until the next governance param update rewrites history from live. It does not halt or diverge — the chain stays self-consistent on the stale value — it just ignores the change. `v0.1.34.go` got this right by pairing its stamp with `SetParamsAtHeight`; `v0.0.10`/`v0.0.13`/`v0.1.13` predate history and did not. If a future handler must change a shared param, write BOTH live and a history entry at the upgrade height.
- **Offchain pricing and window timing must track the onchain epoch** - The RelayMiner re-derives claim value and window boundaries locally: to decide whether a proof is worth submitting (`pkg/relayer/session/proof.go`), to meter overservicing (`pkg/relayer/proxy/relay_meter.go`), to gate reward eligibility (`pkg/relayer/relay_authenticator/relay_verifier.go`), and to close websocket bridges (`pkg/relayer/proxy/async.go`). All of these read `sharedQueryClient.GetParamsAtHeight` — **pricing at session START, window timing at session END**, mirroring `x/proof`. Using `GetParams` (live) at any of them re-opens the divergence: mis-priced claims skip a proof the chain requires (`PROOF_MISSING` → slash), and mis-timed windows submit claims/proofs outside them. `GetParamsAtHeight` deliberately has no "live already describes this height" shortcut — see its godoc for why the `session_grid_anchor_height` guard was wrong.
- **Watch integer crossings when changing session length (`num_blocks_per_session`)** - Derived values computed with `ceil()` are step functions, so a small `N` change can discontinuously double or halve a derived quantity. Compute the before/after values for realistic inputs rather than assuming the change is proportional.

### Testing Architecture

- **Unit tests** - In `*_test.go` files alongside source
- **Integration tests** - Cross-module testing in `/tests/integration/`, using in-memory blockchain from `testutil/integration/app.go`
- **E2E tests** - Gherkin scenarios in `/e2e/tests/` using LocalNet and `gocuke` (Gherkin BDD)
- **Test utilities** - Mocks and fixtures in `/testutil/`
- **Keeper test factories** - Options pattern in `testutil/keeper/` for isolated keeper testing
- **Build tags** - Tests use build tags: `test`, `integration`, `e2e`, `load`. Ensure the correct tag is set or tests won't be included.

## Protocol Buffer Workflow

1. Modify `.proto` files in `/proto/`
2. Run `make proto_regen` to generate Go code
3. Update keeper methods and message handlers
4. Add/update tests for new functionality
5. Run `make go_lint` before committing

## Chain Upgrade Workflow

Consensus-breaking changes ship through an upgrade handler in `/app/upgrades/`. One file per
release (`v0.1.34.go`, `v0.1.35.go`, ...), each defining an `upgrades.Upgrade` value with a
`PlanName`, `StoreUpgrades`, and `CreateUpgradeHandler`.

1. Copy `app/upgrades/vNEXT.go` (kept in tree as the working next-release handler, with
   `vNEXT_Template.go` as the pristine template) and rename `NEXT` to the release version.
2. Implement the handler. Params added or changed since the previous release MUST be set here -
   diff `vPREV..vNEXT` and cross-check `config.yml`, since a param added to the proto without a
   corresponding handler write is left at its zero value on upgraded chains.
3. **Register it**: append the new `upgrades.Upgrade_X_Y_Z` value to the `allUpgrades` slice in
   `app/upgrades.go`. Defining the descriptor does NOT register it - `setUpgrades` only ranges
   over `allUpgrades`. An unregistered handler HALTS THE CHAIN at the upgrade height:
   `x/upgrade`'s BeginBlocker errors on `!HasHandler(plan.Name)`, which panics the node.
   `app/upgrades_registration_test.go` guards this; keep it passing.
4. The upgrade MUST be in a released binary before the on-chain upgrade is scheduled, so
   `cosmovisor` can fetch it from GitHub Releases.

Older handlers stay in the slice commented out with a one-line note on what they did - follow that
convention rather than deleting them.

Release tagging is scripted in `makefiles/release.mk`: `release_tag_rc`, `release_tag_minor`,
`release_tag_major`, plus `release_tag_dev` / `release_tag_local_testing` for unmerged work.

### Upgrade handler safety

An upgrade handler runs inside consensus, so a panic or an error return halts the chain at the
upgrade height rather than failing a transaction. Two consequences:

- Validate that new param values are legal against the *live* values they will coexist with. A
  handler that writes params which fail the module's `Validate()` will halt the chain.
- Everything under Consensus Safety below applies to handler code too.

## Adding Query Endpoints

To add a new gRPC/REST query endpoint to a module:

1. **Proto definition** (`proto/pocket/<module>/query.proto`):
   - Add RPC method to the `Query` service with `google.api.http` option for REST
   - Add request/response message types
   - For pagination, use `cosmos.base.query.v1beta1.PageRequest/PageResponse`

2. **Regenerate**: `make proto_regen`

3. **Query handler** (`x/<module>/keeper/query_*.go`):
   - Implement the method on the Keeper that matches the generated interface
   - Use `query.Paginate()` for paginated queries over store prefixes
   - Return gRPC status errors (e.g., `status.Error(codes.NotFound, ...)`)

4. **CLI registration** (`x/<module>/module/autocli.go`):
   - Add `RpcCommandOptions` entry with `RpcMethod`, `Use`, `Short`, `Long`, `Example`
   - Use `PositionalArgs` to map CLI args to proto fields

5. **Verify**: `make go_lint && go test ./x/<module>/... && make ignite_build`

## LocalNet Development

Use LocalNet for testing multi-node scenarios and protocol upgrades:

- Configuration in `/localnet/kubernetes/`
- Observability with Grafana dashboards
- Reset network state with `make localnet_reset` when testing breaking changes
