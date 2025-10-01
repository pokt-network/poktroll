# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Poktroll is a Cosmos SDK-based blockchain implementing Pocket Network's Shannon upgrade - a decentralized API layer for Web3. Built with Go 1.24.3, Cosmos SDK v0.53.0, and CometBFT consensus.

## Development Commands

### Core Development

```bash
make go_develop                 # Generate protos and mocks (run after proto changes)
make go_develop_and_test       # Generate + run all tests
make ignite_build_pocketd      # Build pocketd binary
make proto_regen               # Regenerate protobuf artifacts
```

### Testing

```bash
make test_all                  # Run all tests
make test_e2e                  # End-to-end tests with Gherkin scenarios
make test_integration          # Integration tests
make go_lint                   # Run linters (always run before commits)
```

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

### Key Components

- **`/pkg/relayer/`** - RelayMiner implementation for API proxying
- **`/pkg/client/`** - Blockchain client libraries and query helpers
- **`/pkg/crypto/`** - Ring signatures and cryptographic utilities
- **`/pkg/observable/`** - Reactive programming patterns using channels
- **`/pkg/polylog/`** - Structured logging framework (use instead of standard log)

### Development Patterns

- **Protocol-first development** - Always update `.proto` files before implementation
- **Keeper pattern** - State management through module keepers with proper gas metering
- **Event-driven architecture** - Emit typed events for cross-module communication
- **Observable patterns** - Use `pkg/observable` for reactive data flows
- **Ring signatures** - Privacy-preserving authentication in `pkg/crypto/rings`

### Testing Architecture

- **Unit tests** - In `*_test.go` files alongside source
- **Integration tests** - Cross-module testing in `/tests/integration/`
- **E2E tests** - Gherkin scenarios in `/e2e/tests/` using LocalNet
- **Test utilities** - Mocks and fixtures in `/testutil/`

## Protocol Buffer Workflow

1. Modify `.proto` files in `/proto/`
2. Run `make proto_regen` to generate Go code
3. Update keeper methods and message handlers
4. Add/update tests for new functionality
5. Run `make go_lint` before committing

## LocalNet Development

Use LocalNet for testing multi-node scenarios and protocol upgrades:

- Configuration in `/localnet/kubernetes/`
- Observability with Grafana dashboards
- Reset network state with `make localnet_reset` when testing breaking changes
