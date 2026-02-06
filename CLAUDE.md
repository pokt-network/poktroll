# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Poktroll is a Cosmos SDK-based blockchain implementing Pocket Network's Shannon upgrade - a decentralized API layer for Web3. It enables applications to access API services through a network of suppliers (RelayMiners), with cryptographic proofs of service and tokenomic incentives.

- **Language**: Go 1.24.3 (CGO disabled)
- **Framework**: Cosmos SDK v0.53.0, CometBFT consensus
- **Binary**: `pocketd` (the blockchain daemon)
- **Module path**: `github.com/pokt-network/poktroll`
- **Account prefix**: `pokt`
- **Token**: `upokt` (micro-POKT)

## Development Commands

### Core Development

```bash
make go_develop                 # Generate protos and mocks (run after proto changes)
make go_develop_and_test       # Generate + run all tests
make ignite_pocketd_build      # Build pocketd binary to GOPATH/bin
make proto_regen               # Regenerate protobuf artifacts
make go_lint                   # Run linters (always run before commits)
```

### Testing

```bash
make test_all                  # Run all unit tests (build tag: test)
make test_integration          # Integration tests (build tag: integration)
make test_e2e                  # E2E tests with Gherkin scenarios (build tag: e2e, requires LocalNet)
make test_e2e_relay            # Run only relay.feature
make test_e2e_tokenomics       # Run only 0_tokenomics.feature
make test_e2e_app              # Run only stake_app.feature
make test_load_relays_stress_localnet  # Load/stress testing
```

### LocalNet Operations

```bash
make localnet_up               # Start local dev network (Tilt + Kind k8s)
make localnet_down             # Stop local network
make localnet_reset            # Reset and restart network
make localnet_regenesis        # Regenerate genesis state
make acc_balance_query ACC=<addr>  # Query account balance
```

## Directory Structure

```
poktroll/
├── app/                    # Application wiring, keepers, upgrades
│   ├── app.go              # Core app initialization (extends runtime.App)
│   ├── app_config.go       # Cosmos depinject module configuration
│   ├── ante.go             # Transaction ante-handlers
│   ├── ibc.go              # IBC module setup
│   ├── keepers/types.go    # All 27 keepers (SDK + Pocket) in one struct
│   └── upgrades/           # Version-specific upgrade handlers (v0.0.4 → v0.1.31+)
├── cmd/pocketd/            # CLI entry point and command definitions
├── x/                      # Cosmos SDK modules (core blockchain logic)
├── pkg/                    # Shared Go libraries (off-chain and on-chain)
├── proto/pocket/           # Protobuf definitions for all modules
├── api/                    # Generated API code from protos
├── tests/integration/      # Cross-module integration tests
├── e2e/tests/              # Gherkin BDD end-to-end tests (14 .feature files)
├── testutil/               # Test utilities, mocks, fixtures, keeper factories
├── localnet/               # LocalNet config (Kubernetes, Grafana, Docker)
├── load-testing/           # Stress/load test manifests and scenarios
├── tools/scripts/          # Utility scripts (staking, params, IBC)
├── docusaurus/             # Documentation site (dev.poktroll.com)
├── makefiles/              # Modular Makefile includes
├── telemetry/              # Observability and metrics
└── config.yml              # Ignite CLI chain configuration (accounts, genesis)
```

## Architecture

### Core Modules (`/x/`)

Each module follows the Cosmos SDK pattern: `keeper/`, `module/`, `types/`, `simulation/`.

| Module | Purpose | Key Types |
|--------|---------|-----------|
| **application** | App staking, delegation to gateways, service configs | `Application`, `PendingUndelegation` |
| **supplier** | Service provider management, operator/owner separation | `Supplier`, `SupplierServiceConfig` |
| **gateway** | QoS layer for enterprise app access | `Gateway` |
| **service** | API service registry, compute unit pricing, relay mining difficulty | `Service`, `ApplicationServiceConfig` |
| **session** | Time-bounded interaction windows (app ↔ supplier matching) | `Session`, `SessionHeader` |
| **proof** | Cryptographic verification via Sparse Merkle Trees (SMT) | `Claim`, `Proof`, `ClaimProofStatus` |
| **tokenomics** | Settlement, mint/burn economics, token distribution | Token Logic Modules (TLMs) |
| **shared** | Cross-module params (blocks per session, CUPR, etc.) | `Supplier` (proto), `ApplicationServiceConfig` |
| **migration** | Morse → Shannon state migration | `MorseClaimableAccount` |

### Module Interaction Flow

```
Application stakes → Session assigns suppliers → Supplier serves relays →
Supplier creates Claim (SMT root) → Supplier submits Proof (SMT path) →
Tokenomics settles (mint/burn/distribute)
```

### Key Packages (`/pkg/`)

| Package | Purpose |
|---------|---------|
| `pkg/relayer/` | RelayMiner: proxy, session management, miner, SMT persistence |
| `pkg/client/` | Blockchain clients: block, tx, query, events, supplier, delegation |
| `pkg/crypto/rings/` | Ring signatures for privacy-preserving relay authentication |
| `pkg/observable/` | Reactive programming (Observable, ReplayObservable via Go channels) |
| `pkg/polylog/` | Structured logging framework (use instead of standard log) |
| `pkg/cache/` | Cache interfaces (memory, key-value, params caching) |
| `pkg/either/` | Either/Result type pattern for error handling |
| `pkg/retry/` | Retry logic with backoff |
| `pkg/deps/config/` | Dependency injection supplier functions (depinject wiring) |

### Dependency Injection

The project uses Cosmos SDK's `depinject` extensively. Off-chain components (RelayMiner) are wired in `pkg/deps/config/suppliers.go` via `SupplierFn` functions:

```go
// Pattern: each component has a NewSupply*Fn that returns a SupplierFn
type SupplierFn func(context.Context, depinject.Config, *cobra.Command) (depinject.Config, error)
```

Key supplier functions: `NewSupplyBlockClientFn`, `NewSupplyQueryClientContextFn`, `NewSupplyRingClientFn`, `NewSupplySupplierClientsFn`, `NewSupplyRelayerProxyFn`, etc.

## Module Structure Pattern

Each module in `x/<module>/` follows this layout:

```
x/<module>/
├── keeper/
│   ├── keeper.go            # Keeper struct and constructor
│   ├── query.go             # var _ types.QueryServer = Keeper{}
│   ├── query_*.go           # Query handler implementations
│   ├── msg_server_*.go      # Message handler implementations
│   └── *_test.go            # Unit tests
├── module/
│   ├── module.go            # AppModule interface implementation
│   ├── autocli.go           # AutoCLI configuration (RpcCommandOptions)
│   ├── query.go             # CLI query command registration
│   ├── tx.go                # CLI tx command registration
│   └── helpers.go           # Module-specific helpers
├── types/
│   ├── keys.go              # Store keys and prefixes
│   ├── params.go            # Module parameters and validation
│   ├── message_*.go         # Message type definitions and validation
│   ├── errors.go            # Module-specific error codes
│   ├── events.go            # Event type definitions
│   ├── genesis.go           # Genesis state validation
│   └── *.pb.go              # Generated protobuf types
└── simulation/              # State simulation for fuzz testing
```

## Protocol Buffer Workflow

Proto files live in `proto/pocket/<module>/` with these standard files per module:
- `query.proto` - Query service RPCs with `google.api.http` annotations
- `tx.proto` - Transaction message types
- `types.proto` - Core data structures
- `params.proto` - Module parameters
- `event.proto` - Typed events
- `genesis.proto` - Genesis state
- `module/module.proto` - Cosmos app module config

**Code generation** uses `buf` with gogo-proto:
- `proto/buf.gen.gogo.yaml` - Go code generation
- `proto/buf.gen.sta.yaml` - OpenAPI/Swagger generation
- `proto/buf.gen.ts.yaml` - TypeScript generation

**Workflow:**
1. Modify `.proto` files in `proto/pocket/<module>/`
2. Run `make proto_regen` to generate Go code into `api/` and `x/<module>/types/`
3. Update keeper methods to implement generated interfaces
4. Add/update tests
5. Run `make go_lint` before committing

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

5. **Verify**: `make go_lint && go test ./x/<module>/... && make ignite_pocketd_build`

## Testing Architecture

### Build Tags

Tests are controlled by build tags:
- `//go:build test` - Unit tests (default with `make test_all`)
- `//go:build integration` - Integration tests (`make test_integration`)
- `//go:build e2e` - End-to-end tests (`make test_e2e`, requires LocalNet)
- `//go:build load` - Load/stress tests

### Unit Tests

Standard Go tests alongside source code. Use `testutil/keeper/` factories to create isolated module keepers:

```go
keepers, ctx := keeper.NewTokenomicsModuleKeepers(t, log.NewNopLogger(),
    keeper.WithProofRequirement(false),
    keeper.WithDefaultModuleBalances(),
)
```

### Integration Tests (`/tests/integration/`)

Cross-module tests using in-memory blockchain (`testutil/integration/app.go` → `App` struct). No LocalNet required. Subdirectories: application, supplier, service, tokenomics, params, migration.

### E2E Tests (`/e2e/tests/`)

Gherkin BDD scenarios using `gocuke` framework against a running LocalNet:
- 14 `.feature` files covering relay lifecycle, staking, sessions, tokenomics, params, migrations
- Step implementations in `*_test.go` files
- Runs `pocketd` CLI commands via shell
- Tag filtering: `@manual`, `@oneshot` for special scenarios

### Test Utilities (`/testutil/`)

- `testutil/keeper/` - Per-module keeper factories with options pattern
- `testutil/integration/` - Full in-memory blockchain app + reusable test suites
- `testutil/testkeyring/` - Pre-generated deterministic test accounts
- `testutil/sample/` - Random address generators
- `testutil/testproof/` - Proof/claim fixtures
- `testutil/testclient/` - Real client wrappers for testing
- Mock directories: `mockclient/`, `mockrelayer/`, `mockcrypto/`

## Upgrade System

Upgrades are in `app/upgrades/` with handlers registered in `app/upgrades.go`:

```go
var allUpgrades = []upgrades.Upgrade{
    upgrades.Upgrade_0_1_31_beta_2,
    // ... previous versions commented out
}
```

To add a new upgrade:
1. Create `app/upgrades/vX.Y.Z.go` with an `Upgrade` variable
2. Add to `allUpgrades` slice in `app/upgrades.go`
3. Handler can modify state, add params, migrate data at the scheduled block height
4. Cosmovisor automatically pulls new binary from GitHub releases

## LocalNet Development

- **Orchestration**: Tilt + Kind (Kubernetes in Docker)
- **Config**: `config.yml` (Ignite chain config with 20+ test accounts)
- **Hot Reloading**: Changes in `app/`, `cmd/`, `x/`, `pkg/` trigger validator restart
- **Observability**: Prometheus + Grafana with 16 custom dashboards in `localnet/grafana-dashboards/`
- **Genesis**: Pre-configured with services (anvil, rest, anvilws, ollama), staked apps/suppliers/gateways
- **Session params**: 10 blocks per session (LocalNet); production uses longer sessions

Reset network state with `make localnet_reset` when testing breaking changes.

## Consensus Safety (Critical)

Cosmos SDK blockchains require **deterministic execution** - all validators must produce identical state from identical inputs. Non-determinism causes AppHash mismatches and chain halts.

### Forbidden Patterns in Keeper Code

**Never do these in message handlers, BeginBlock, or EndBlock:**

1. **In-memory caches on Keeper structs** - Different nodes have different cache states based on query history, causing gas mismatches
   ```go
   // BAD: Cache populated by queries affects tx execution gas
   type Keeper struct {
       sessionCache map[string]*Session  // NEVER DO THIS
   }
   ```

2. **Map iteration without sorting** - Go map iteration order is randomized
   ```go
   // BAD: Non-deterministic order
   for k, v := range myMap { ... }

   // GOOD: Sort keys first
   keys := maps.Keys(myMap)
   slices.Sort(keys)
   for _, k := range keys { ... }
   ```

3. **time.Now() or rand** - Use `ctx.BlockTime()` and deterministic randomness from block hash

4. **External API/RPC calls** - Network responses vary between nodes

5. **Goroutines without deterministic synchronization** - Results must be identical regardless of execution order

### Safe Patterns

- **Ephemeral per-block caches** - Create fresh in BeginBlock/EndBlock, don't persist on Keeper
- **Store-backed caches** - Part of consensus state, identical across nodes
- **Query-only caches** - Only cache during `ExecModeCheck`/`ExecModeSimulate`, never during `ExecModeFinalize`
- **Sorted iteration** - Always sort map keys before iterating when order affects state

### Why LocalNet Won't Catch These Bugs

In-memory cache bugs are insidious because:
- Single-node testing always passes (one cache state)
- Multi-node passes if all nodes receive identical query traffic
- Only fails in production when validators have divergent cache states from different RPC query patterns
- Manifests as "gas mismatch" or "AppHash mismatch" which can be misattributed

**Always review keeper struct fields for consensus safety when adding caching or optimization.**

## Key Domain Concepts

- **Session**: A time-bounded window (N blocks) pairing an application with a set of suppliers for a specific service. Session ID is deterministically derived from app address, service ID, and block height.
- **Claim**: After a session, a supplier submits a claim containing the SMT root hash of all relays served. This is the "I did work" assertion.
- **Proof**: A cryptographic proof (sparse Merkle tree closest path) that validates the claim. Required probabilistically or when claim value exceeds a threshold.
- **Settlement**: Tokenomics processes claims/proofs to mint/burn tokens and distribute rewards to suppliers, the DAO, service owners, and proposers.
- **RelayMiner**: Off-chain component (`pkg/relayer/`) that proxies API requests, builds SMTs, and submits claims/proofs to the chain.
- **Ring Signatures**: Applications sign relays using ring signatures (via `pkg/crypto/rings/`) so suppliers can verify the relay came from a valid app without knowing which specific app.
- **Compute Units**: Abstract unit measuring API service cost. Each relay has a `compute_units_per_relay` (CUPR) value. Settlement converts compute units to tokens via `compute_units_to_tokens_multiplier`.

## Code Conventions

### TODO Tags

The codebase uses prefixed TODO tags to categorize technical debt:

- `TODO_TECHDEBT` - Known tech debt to address later
- `TODO_MAINNET` - Must be resolved before mainnet launch
- `TODO_POST_MAINNET` - Can wait until after mainnet
- `TODO_UPNEXT` - Prioritized for upcoming work
- `TODO_IMPROVE` - Non-critical improvements
- `TODO_CONSIDERATION` - Design decisions to revisit
- `TODO_URGENT` - High-priority issues

Format: `TODO_TAG(@assignee, #issue): Description`

### Error Definitions

Each module defines sentinel errors in `x/<module>/types/errors.go`:
```go
import sdkerrors "cosmossdk.io/errors"

var (
    ErrAppNotFound     = sdkerrors.Register(ModuleName, 1104, "application not found")
    ErrAppInvalidStake = sdkerrors.Register(ModuleName, 1101, "invalid application stake")
)
```
Error codes are module-scoped (e.g., application uses 11xx, proof uses different range). Always use unique codes within a module.

### Expected Keepers (Cross-Module Dependencies)

Modules declare the interfaces they need from other modules in `x/<module>/types/expected_keepers.go`:
```go
//go:generate go run go.uber.org/mock/mockgen -destination ../../../testutil/<module>/mocks/expected_keepers_mock.go -package mocks . AccountKeeper,BankKeeper

type BankKeeper interface {
    SpendableCoins(context.Context, sdk.AccAddress) sdk.Coins
    SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
}
```
After changing expected keepers, run `go generate ./x/<module>/types/` to regenerate mocks.

### Event Emission

Emit typed events from keeper methods:
```go
sdkCtx := sdk.UnwrapSDKContext(ctx)
if err = sdkCtx.EventManager().EmitTypedEvents(&types.EventApplicationStaked{
    Application: &app,
}); err != nil {
    return nil, types.ErrAppEmitEvent.Wrapf("(%+v): %s", event, err)
}
```
Event types are defined as protobuf messages in `proto/pocket/<module>/event.proto`.

### Telemetry

Message handlers use deferred telemetry counters:
```go
isSuccessful := false
defer telemetry.EventSuccessCounter("stake_application", telemetry.DefaultCounterFn, func() bool { return isSuccessful })
// ... handler logic ...
isSuccessful = true
```

## Implementing a New Message Handler

1. **Proto definition** (`proto/pocket/<module>/tx.proto`):
   - Add RPC method to `Msg` service
   - Define `MsgYourAction` and `MsgYourActionResponse` message types
   - Add `cosmos.msg.v1.signer` option

2. **Regenerate**: `make proto_regen`

3. **Message validation** (`x/<module>/types/message_your_action.go`):
   - Implement `ValidateBasic()` on the message type

4. **Handler** (`x/<module>/keeper/msg_server_your_action.go`):
   - Implement the method on `msgServer`
   - Pattern: validate → read state → business logic → write state → emit events
   - Return gRPC status errors: `status.Error(codes.InvalidArgument, err.Error())`
   - Use `logger := k.Logger().With("method", "YourAction")`

5. **Tests** (`x/<module>/keeper/msg_server_your_action_test.go`)

6. **CLI registration** (`x/<module>/module/autocli.go`):
   - Add `RpcCommandOptions` entry in `TxAutoConfig`

7. **Verify**: `make go_lint && go test ./x/<module>/...`

## Adding a New Module Parameter

1. **Proto** (`proto/pocket/<module>/params.proto`):
   - Add field to `Params` message
   - If supporting single-param updates, add to `MsgUpdateParam` oneof `as_type`

2. **Regenerate**: `make proto_regen`

3. **Param constants and defaults** (`x/<module>/types/params.go`):
   - Add `ParamYourParam = "your_param"` constant
   - Add `DefaultYourParam` default value
   - Add `ValidateYourParam()` function
   - Update `NewParams()`, `DefaultParams()`, and `Validate()` (calls `ValidateYourParam`)

4. **MsgUpdateParam handler** (`x/<module>/keeper/msg_server_update_param.go`):
   - Add case to the switch statement:
   ```go
   case types.ParamYourParam:
       params.YourParam = msg.GetAsType()
   ```

5. **Genesis and migration**: Update default genesis if needed

6. **Tests**: Add param validation tests and update param test

7. **Verify**: `make go_lint && go test ./x/<module>/...`

## EndBlocker / BeginBlocker Pattern

```go
// x/<module>/module/abci.go
func EndBlocker(ctx sdk.Context, k keeper.Keeper) (err error) {
    defer cosmostelemetry.ModuleMeasureSince(types.ModuleName, cosmostelemetry.Now(), cosmostelemetry.MetricKeyEndBlocker)
    logger := k.Logger().With("method", "EndBlocker")
    // ... business logic ...
    return nil
}
```
Registered in `x/<module>/module/module.go` via `appmodule.HasEndBlocker` interface.
