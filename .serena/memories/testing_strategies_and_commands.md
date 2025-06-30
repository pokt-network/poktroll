# Testing Strategies and Commands

## Testing Architecture Levels

### Unit Tests
- **Location**: `*_test.go` files alongside source code
- **Command**: `make test_all` - Run all Go tests (default, fastest)
- **Command**: `make test_verbose` - Verbose output with race detection
- **Scope**: Individual functions, methods, and components

### Integration Tests
- **Location**: `/tests/integration/` directory
- **Command**: `make test_integration` - In-memory integration tests only
- **Command**: `make test_all_with_integration` - All tests including integration
- **Scope**: Cross-module interactions, keeper integrations

### End-to-End (E2E) Tests
- **Location**: `/e2e/tests/` directory with Gherkin `.feature` files
- **Command**: `make test_e2e` - Run all E2E tests against LocalNet
- **Command**: `make test_e2e_verbose` - E2E with debug output
- **Environment**: Requires LocalNet running (`make localnet_up`)

## Specific E2E Test Suites

### Module-Specific E2E Tests
- `make test_e2e_app` - Application lifecycle (staking/unstaking)
- `make test_e2e_supplier` - Supplier lifecycle and operations
- `make test_e2e_gateway` - Gateway lifecycle and delegation
- `make test_e2e_session` - Session management and claim/proof lifecycle
- `make test_e2e_relay` - Complete relay request/response flow
- `make test_e2e_tokenomics` - Settlement and economic calculations
- `make test_e2e_params` - Parameter updates across all modules

### Load Testing
- **Location**: `/load-testing/tests/` directory
- `make test_load_relays_stress_localnet` - LocalNet stress testing
- `make test_load_relays_stress_custom` - Custom manifest testing
- Uses YAML manifest files in `/load-testing/` for configuration

## Testing Environment Setup

### E2E Test Environment Variables
- `POCKET_NODE=$(POCKET_NODE)` - Validator endpoint
- `PATH_URL=$(PATH_URL)` - API gateway URL  
- `POCKETD_HOME=../../$(POCKETD_HOME)` - LocalNet home directory
- `E2E_DEBUG_OUTPUT=true` - Enable verbose E2E debugging

### Test Tags and Build Modes
- `-tags=e2e,test` - E2E test inclusion
- `-tags=test,integration` - Integration test inclusion
- `-tags=load,test` - Load test inclusion
- `-buildmode=pie` - macOS compatibility (excludes race detection)
- `-race` - Race condition detection (not compatible with buildmode=pie)

## Quality Assurance Commands

### Linting and Code Quality
- `make go_lint` - **ALWAYS run before commits**
- `make install_ci_deps` - Install golangci-lint and development tools

### Iterative Testing
- `make itest` - Run tests iteratively with custom patterns
- `./tools/scripts/itest.sh` - Underlying iterative test script

### Test Fixture Generation
- `make test_gen_fixtures` - Verify fixture generation in `pkg/relayer/miner/gen/`

## Best Practices

1. **Always run linting**: `make go_lint` before any commit
2. **Protocol-first development**: Update `.proto` files first, then run `make proto_regen`
3. **LocalNet for E2E**: Use `make localnet_reset` to ensure clean state
4. **Test categorization**: Use appropriate build tags for test inclusion
5. **Flaky test handling**: Set `INCLUDE_FLAKY_TESTS=true` when needed