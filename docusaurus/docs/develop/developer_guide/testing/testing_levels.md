---
sidebar_position: 1
title: Testing Levels
---

## Testing Levels <!-- omit in toc -->

### Table Of Contents

- [Table Of Contents](#table-of-contents)
- [Unit Tests](#unit-tests)
- [Module Integration Tests](#module-integration-tests)
  - [Unit Test Example](#unit-test-example)
  - [Unit Test - Good Fit](#unit-test---good-fit)
  - [Unit Test - Bad Fit](#unit-test---bad-fit)
  - [Unit Test - Limitations](#unit-test---limitations)
- [App Integration Tests](#app-integration-tests)
  - [Integration Test Example](#integration-test-example)
  - [Integration Test 0 Good Fit](#integration-test-0-good-fit)
  - [Integration Test 0 Bad Fit](#integration-test-0-bad-fit)
  - [Integration Test - Limitations](#integration-test---limitations)
- [In-Memory Network Integration Tests](#in-memory-network-integration-tests)
  - [In-Memory Network Example](#in-memory-network-example)
  - [In-Memory Network - Good Fit](#in-memory-network---good-fit)
  - [In-Memory Network - Bad Fit](#in-memory-network---bad-fit)
  - [In-Memory Network - Limitations](#in-memory-network---limitations)
- [End-to-End Tests](#end-to-end-tests)
  - [E2E Test Example](#e2e-test-example)
  - [E2E Test - Good Fit](#e2e-test---good-fit)
  - [E2E Test - Bad Fit](#e2e-test---bad-fit)
  - [E2E Test - Limitations](#e2e-test---limitations)

### Unit Tests

**Unit tests** are the most granular level of testing, focusing on individual functions or methods within a module or module subcomponent.
These tests are used to verify that each unit of code behaves as expected in isolation.

### [Module Integration Tests](module_integration.md)

**Module integration tests** focus on testing the interaction between modules in the appchain without mocking individual components.
This level of testing ensures that cross-module interactions can be exercised but without the overhead of the full appchain.

#### Unit Test Example

```go
// TODO_DOCUMENT(@bryanchriswhite): Add example
```

#### Unit Test - Good Fit

- Exercising a `Keeper` method
- Code has dependencies on other module `Keeper`s

#### Unit Test - Bad Fit

- Test depends on network events
- Test depends on `Tx` assertions

#### Unit Test - Limitations

- No transactions
- No events
- No message server

### [App Integration Tests](app_integration)

**App integration tests** focus on testing the behavior of the fully integrated _appchain_ from a common message server interface.
This level of testing ensures message handling logic is exercise while fully integrated with cosmos-sdk but without the overhead of the cometbft engine and networking.

#### Integration Test Example

```go
// TODO_DOCUMENT(@bryanchriswhite): Add example
```

_NOTE: See [App Integration Suites](integration_suites) for organizing larger or higher-level app integration tests._

#### Integration Test 0 Good Fit

- Exercising a user story involving multiple messages
- Exercising a scenario involving multiple messages
- Exercising cross-module dependencies & interactions
- Asserting against a new integrated state

#### Integration Test 0 Bad Fit

- Code under test depends/asserts on networking operations
- Code under test depends/asserts on consensus operations
- Code under test requires setup which would be simpler to do with direct keeper interaction.

#### Integration Test - Limitations

- No networking
- No consensus
- No keeper API access (intentional)

### [In-Memory Network Integration Tests](in_memory_integration)

**In-memory network integration tests** focus on testing the behavior of a multi-validator network from the perspective of the ABCI message interface.
This level of testing ensures that the appchain behaves as expected in a multi-validator environment.

#### In-Memory Network Example

```go
// TODO_DOCUMENT(@bryanchriswhite): Add example
```

#### In-Memory Network - Good Fit

- Exercising CometBFT RPC
- Exercising consensus scenarios
- Exercising multi-validator scenarios
- Integrating with external tools via network

#### In-Memory Network - Bad Fit

- Most cases; use sparingly
- Prefer other levels unless it's clearly appropriate

#### In-Memory Network - Limitations

- No parallelization
- No `Keeper` module access
- No API access
- Depends on cosmos-sdk APIs (less customizable)
- Slow startup time (per network).

### [End-to-End Tests](e2e)

**End-to-end tests** focus on testing the behavior of a network containing both on- and off-chain actors; typically exercising "localnet".

#### E2E Test Example

```go
// TODO_DOCUMENT(@bryanchriswhite): Add example
```

#### E2E Test - Good Fit

- Asserts or dependent on off-chain assertions
- Asserts or dependent on off-chain actors
- Asserts or dependent on off-chain behavior

#### E2E Test - Bad Fit

- Scenarios which require many blocks/sessions to complete
- Scenarios which are not idempotent
- Scenarios which assume specific/complex network states

#### E2E Test - Limitations

- Depends on LocalNet to be running and healthy
- Depends on other environments (DevNet/TestNet) to be running and healthy
- Shared mutable network state on-chain
- Shared mutable network state off-chain
- Intolerant of non-idempotent operations (CI re-runnability).
