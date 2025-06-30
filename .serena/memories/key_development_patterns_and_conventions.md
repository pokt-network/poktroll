# Key Development Patterns and Conventions

## Development Philosophy

### Protocol-First Development
- **Always update `.proto` files before implementation**
- Generate Go code with `make proto_regen` after proto changes
- Proto files in `/proto/pocket/<module>/` define the contract
- Generated code appears in `/api/pocket/<module>/`

### Keeper Pattern
- **State management through module keepers with proper gas metering**
- Each module has a keeper in `x/<module>/keeper/`
- Keepers handle state reads/writes, business logic validation
- Proper gas metering for all state operations

## Key Libraries and Patterns

### Logging Framework
- **Use `pkg/polylog` instead of standard Go log**  
- Structured logging with context support
- Example: `logger := polylog.Ctx(ctx)`
- Provides consistent logging across all components

### Observable Patterns
- **Use `pkg/observable` for reactive data flows**
- Channel-based reactive programming
- Located in `pkg/observable/channel/`
- Preferred for event-driven architectures

### Cryptographic Utilities
- **Ring signatures** - Privacy-preserving authentication in `pkg/crypto/rings`
- **Protocol crypto** - Hash utilities in `pkg/crypto/protocol/`
- Relay difficulty calculations and proof path generation

### Client Libraries
- **Blockchain clients** - `pkg/client/` for query and transaction operations
- **Query clients** - Typed query interfaces in `pkg/client/query/`
- **Transaction clients** - Transaction building in `pkg/client/tx/`

## Event-Driven Architecture

### Event Emission
- **Emit typed events for cross-module communication**
- Events defined in `proto/pocket/<module>/event.proto`
- Generated event types in `api/pocket/<module>/event.pulsar.go`
- Use `sdk.EventManager` for event emission in keepers

### Event Handling
- Events consumed by telemetry system (`/telemetry/`)
- Observable patterns for event streaming
- Event replay capabilities in `pkg/client/events/`

## Testing Patterns

### Mock Generation
- **Generated mocks** in `/testutil/<module>/mocks/`
- Use `make go_develop` to regenerate mocks
- Mock interfaces for external dependencies

### Test Utilities
- **Fixtures and helpers** in `/testutil/` directory structure
- Module-specific test utilities in `/testutil/<module>/`
- Network simulation in `/testutil/network/`
- Sample data generation in `/testutil/sample/`

### Integration Test Suites
- **Base test suites** in `/testutil/integration/suites/`
- Each module has dedicated test suite (e.g., `suites/application.go`)
- Cross-module integration testing support

## Code Organization Conventions

### Package Structure
- `/pkg/` - Reusable libraries and utilities
- `/x/` - Cosmos SDK modules (blockchain business logic)
- `/cmd/` - Command-line interfaces and main executables
- `/app/` - Application initialization and configuration

### RelayMiner Implementation
- **Location**: `/pkg/relayer/` - Complete RelayMiner implementation
- Proxy capabilities in `/pkg/relayer/proxy/`
- Session management in `/pkg/relayer/session/`
- Mining logic in `/pkg/relayer/miner/`

### Configuration Management
- **RelayMiner configs** - `/pkg/relayer/config/` for configuration parsing
- **LocalNet configs** - `/localnet/pocketd/config/` for development
- YAML-based configuration with validation

## Performance and Reliability

### Caching Strategies
- **Memory caching** - `pkg/cache/memory/` for in-memory caches
- Historical key-value caching with `historical_kvcache.go`
- Query result caching in client libraries

### Synchronization Utilities
- **Limiters and controls** - `pkg/sync2/` for concurrency management
- Custom synchronization primitives
- Rate limiting and backpressure handling

## Security Best Practices

### Ring Signatures
- Privacy-preserving authentication for relay requests
- Implementation in `pkg/crypto/rings/`
- Client interfaces for signature generation/verification

### Cryptographic Proofs
- Relay proof generation and verification
- Hash-based proof paths in `pkg/crypto/protocol/`
- Difficulty adjustment for relay mining

## LocalNet Development Practices

### Configuration Files
- **Kubernetes manifests** - `/localnet/kubernetes/` for container orchestration
- **Grafana dashboards** - `/localnet/grafana-dashboards/` for observability
- **Validator configs** - `/localnet/pocketd/config/` for node setup

### Development Workflow
1. Start LocalNet: `make localnet_up`
2. Make code changes with proto updates first
3. Regenerate code: `make proto_regen` 
4. Run tests: `make test_all`
5. Run linting: `make go_lint` (mandatory before commits)
6. Reset network if needed: `make localnet_reset`