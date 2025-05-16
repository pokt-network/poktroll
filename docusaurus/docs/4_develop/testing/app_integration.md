---
sidebar_position: 3
title: App Integration Tests
---

// TODO(@bryanchriswhite): Replace github source links with godocs links once available.

## Table Of Contents <!-- omit in toc -->

- [Overview](#overview)
- [Using `integration.App`](#using-integrationapp)
  - [Constructors](#constructors)
    - [Customizing `integration.App` Configuration](#customizing-integrationapp-configuration)
  - [Module Configuration](#module-configuration)
    - [Setting Module Genesis State](#setting-module-genesis-state)
  - [Message / Transaction / Block Processing](#message--transaction--block-processing)
- [Example Test](#example-test)

## Overview

[**App integration level**](testing_levels#app-integration-tests) tests leverage a custom construction of the pocket appchain (for testing only).

This construction integrates all the pocket modules (and their cosmos-sdk dependencies) and exercises the appchain's message routing/handling and transaction processing logic.

Tests in this level conventionally use the `testutil/integration` package's `App` structure and constructors to set up the appchain, execute messages, and make assertions against the resulting appchain state.

:::info
See [App Integration Suites](integration_suites) for organizing larger or higher-level app integration tests.
:::

## Using `integration.App`

### Constructors

To create a new instance of the `IntegrationApp` for your tests, use the `NewCompleteIntegrationApp` constructor, which handles the setup of all modules, multistore, base application, etc.:

```go
// NewCompleteIntegrationApp creates a new instance of the App, abstracting out
// all the internal details and complexities of the application setup.
func NewCompleteIntegrationApp(t *testing.T, opts ...IntegrationAppOptionFn) *App

// IntegrationAppOptionFn is a function that receives and has the opportunity to
// modify the IntegrationAppConfig. It is intended to be passed during integration
// App construction to modify the behavior of the integration App.
type IntegrationAppOptionFn func(*IntegrationAppConfig)
```

If more granular control over the application configuration is required, the more verbose `NewIntegrationApp` constructor exposes additional parameters:

```go
// NewIntegrationApp creates a new instance of the App with the provided details
// on how the modules should be configured.
func NewIntegrationApp(
    t *testing.T,
    sdkCtx sdk.Context,
    cdc codec.Codec,
    txCfg client.TxConfig,
    registry codectypes.InterfaceRegistry,
    bApp *baseapp.BaseApp,
    logger log.Logger,
    authority sdk.AccAddress,
    modules map[string]appmodule.AppModule,
    keys map[string]*storetypes.KVStoreKey,
    msgRouter *baseapp.MsgServiceRouter,
    queryHelper *baseapp.QueryServiceTestHelper,
    opts ...IntegrationAppOptionFn,
) *App {
```

#### Customizing `integration.App` Configuration

If the existing [`IntegrationAppConfig`](https://github.com/pokt-network/poktroll/blob/main/testutil/integration/options.go#L13) is insufficient, it may be extended with additional fields, corresponding logic, and `IntegrationAppOptionFn`s to set them.

### Module Configuration

Integrated modules can be configured using `IntegrationAppOptionFn` typed option functions.

Example use cases include:

- Setting custom genesis states one or more modules (see [`integration.WithModuleGenesisState()`](https://github.com/pokt-network/poktroll/blob/main/testutil/integration/options.go#L40)).
- Setting up a faucet account (see: [`newFaucetInitChainerFn()`](https://github.com/pokt-network/poktroll/blob/main/testutil/integration/app.go#L985)).
- Collecting module info (see: [`newInitChainerCollectModuleNames()`](https://github.com/pokt-network/poktroll/blob/main/testutil/integration/suites/base.go#L157)).

#### Setting Module Genesis State

```go
supplierGenesisState := &suppliertypes.GenesisState{
    // ...
}
app := NewCompleteIntegrationApp(t,
    WithModuleGenesisState[suppliermodule.AppModule](supplierGenesisState),
)
```

### Message / Transaction / Block Processing

The `IntegrationApp` provides several methods to manage the lifecycle of transactions and blocks during tests:

- `RunMsg`/`RunMsgs`: Processes one or more messages by:
  - calling their respective handlers
  - packaging them into a transaction
  - finalizing the block
  - committing the state
  - advancing the block height
  - returning the message responses
- `NextBlock`/`NextBlocks`: Only advances the blockchain state to subsequent blocks.

## Example Test

Here's a simple example of how to create a new integration app instance and run a message using the helper functions:

```go
func TestAppIntegrationExample(t *testing.T) {
    // Initialize a new complete integration app with default options.
    app := NewCompleteIntegrationApp(t)

    // Example message to be processed
    msg := banktypes.NewMsgSend(fromAddr, toAddr, sdk.NewCoins(sdk.NewInt64Coin("upokt", 100)))

    // Run the message in the integration app
    res, err := app.RunMsg(t, msg)
    require.NoError(t, err)

    // Check the result
    require.NotNil(t, res, "Expected a valid response for the message")

    // Type assert the result to the message response type
    sendRes, ok := res.(*banktypes.MsgSendResponse)

    require.True(t, ok)
    require.NotNil(t, sendRes)
}
```

This example initializes the app, processes a bank message, and validates the result.
