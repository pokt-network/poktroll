---
sidebar_position: 1
title: Testing Levels
---

# Testing Levels <!-- omit in toc -->

## Table Of Contents

- [Unit Tests](#unit-tests)
- [Module Integration Tests](#module-integration-tests)
- [App Integration Tests](#app-integration-tests)
- [In-Memory Network Integration Tests](#in-memory-network-integration-tests)
- [End-to-End Tests](#end-to-end-tests)


## Unit Tests

**Unit tests** are the most granular level of testing, focusing on individual functions or methods within a module or module subcomponent.
These tests are used to verify that each unit of code behaves as expected in isolation.

## [Module Integration Tests](module_integration.md)

**Module integration tests** focus on testing the interaction between modules in the appchain without mocking individual components.
This level of testing ensures that cross-module interactions can be exercised but without the overhead of the full appchain.

### Example

```go
// TODO_DOCUMENT(@bryanchriswhite): Add example
```

### Good Fit

- Exercising a keeper method which has dependencies on other modules' keepers and their state.

### Bad Fit

Code under test depends or asserts on transaction results or network events.

### Limitations

- No transactions
- No events
- No message server

## [App Integration Tests](app_integration)

**App integration tests** focus on testing the behavior of the fully integrated appchain from a common message server interface.
This level of testing ensures message handling logic is exercise while fully integrated with cosmos-sdk but without the overhead of the cometbft engine and networking.

### Example

```go
// TODO_DOCUMENT(@bryanchriswhite): Add example
```

_NOTE: See [App Integration Suites](integration_suites) for organizing larger or higher-level app integration tests._

### Good Fit

- Exercising a user story involving multiple messages bound for potentially different modules to arrive at and assert against some new integrated state.

### Bad Fit

- Code under test depends or asserts on networking or consensus operations.
- Code under test requires setup which would be simpler to do with direct keeper interaction.

### Limitations

- No networking.
- No consensus.
- No keeper API access (intentional).


## [In-Memory Network Integration Tests](in_memory_integration)

**In-memory network integration tests** focus on testing the behavior of a multi-validator network from the perspective of the ABCI message interface.
This level of testing ensures that the appchain behaves as expected in a multi-validator environment.

### Example

```go
// TODO_DOCUMENT(@bryanchriswhite): Add example
```

### Good Fit

- Exercising cometbft RPC.
- Exercising consensus/multi-validator scenarios.
- Integrating with external tools via network.

### Bad Fit

- Most cases; prefer other levels unless it's clearly appropriate.

### Limitations

- No parallelization.
- No keeper or module API access.
- Depends on cosmos-sdk APIs (less customizable).
- Slow startup time (per network).


## [End-to-End Tests](e2e)

**End-to-end tests** focus on testing the behavior of a network containing both on- and off-chain actors; typically exercising "localnet".

### Example

```go
// TODO_DOCUMENT(@bryanchriswhite): Add example
```

### Good Fit

- Scenarios which depend or assert on off-chain actor state or behavior

### Bad Fit

- Scenarios which require many blocks or multiple sessions to complete
- Scnearios which are not idempotent
- Scenarios which assume specific and complex network states

### Limitations

- Depends on localnet (or other environment) to be running and healthy.
- Shared mutable network state on- and off-chain.
- Intolerant of non-idempotent operations (CI re-runnability).
