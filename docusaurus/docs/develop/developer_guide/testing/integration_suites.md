---
sidebar_position: 5
title: App Integration Suites
---

// TODO(@bryanchriswhite): Replace github source links with godocs links once available.

## Table of Contents <!-- omit in toc -->

- [Overview](#overview)
- [When to Use Test Suites](#when-to-use-test-suites)
- [Using an Existing Integration Suite](#using-an-existing-integration-suite)
  - [Example (`ParamsSuite`)](#example-paramssuite)
- [Implementing a Test Suite](#implementing-a-test-suite)
  - [Test Suite Gotchas](#test-suite-gotchas)

## Overview

The [`suites` package](https://github.com/pokt-network/poktroll/tree/main/testutil/integration/suites) provides interfaces and base implementations for creating and managing **app integration test** suites.

The foundational components are:

- [**`IntegrationSuite`**](https://github.com/pokt-network/poktroll/blob/main/testutil/integration/suites/interface.go#L14): An interface defining common methods for interacting with an integration app.
- [**`BaseIntegrationSuite`**](https://github.com/pokt-network/poktroll/blob/main/testutil/integration/suites/base.go#L26): A base implementation of the `IntegrationSuite` interface that can be extended by embedding in other test suites.

## When to Use Test Suites

- **Complex Integration Tests**: Testing interactions between several modules; suites facilitate encapsulation and decomposition.
- **Complex Scenarios**: Simulating real-world scenarios that involve several transactions, state changes, and/or complex assertion logic.
- **Reusable Components**: To DRY (Don't Repeat Yourself) up common test helpers which can be embedded in other test suites (object oriented).

## Using an Existing Integration Suite

The `testutil/integration/suites` package contains multiple **app integration suites** which are intended to be embedded in [**app integration level**](testing_levels#app-integration-tests) test suites.

### Example (`ParamsSuite`)

The following example shows a test suite which embeds `suites.ParamsSuite`, in order to set on-chain module params as part of its `SetupTest()` method:

```go
package suites

import (
  "testing"
  "github.com/stretchr/testify/require"
  "github.com/stretchr/testify/suite"
  cosmostypes "github.com/cosmos/cosmos-sdk/types"
  banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

type ExampleTestSuite struct {
  suites.ParamsSuite
}

// SetupTest is called before each test method in the suite.
func (s *ExampleTestSuite) SetupTest() {
  // Initialize a new app instance for each test.
  s.app = NewApp(s.T())

  // Setup the authz accounts and grants for updating parameters.
  s.SetupTestAuthzAccounts()
  s.SetupTestAuthzGrants()

  // Set the module params using the ParamsSuite.
  s.RunUpdateParam(s.T(),
    sharedtypes.ModuleName,
    string(sharedtypes.KeyNumBlocksPerSession),
    9001,
  )
}

func (s *ExampleTestSuite) TestExample() {
  // Query module params using the ParamsSuite.
  sharedParams, err := s.QueryModuleParams(s.T(), sharedtypes.ModuleName)
  require.NoError(s.T(), err)

  // Utilize other BaseIntegrationSuite methods to interact with the app...

  fundAmount := int64(1000)
  fundAddr, err := cosmostypes.AccAddressFromBech32("cosmos1exampleaddress...")
  require.NoError(s.T(), err)

  // Fund an address using the suite's FundAddress method.
  s.FundAddress(s.T(), fundAddr, fundAmount)

  // Use the bank query client to verify the balance.
  bankQueryClient := s.GetBankQueryClient()
  balRes, err := bankQueryClient.Balance(s.SdkCtx(), &banktypes.QueryBalanceRequest{
      Address: fundAddr.String(),
      Denom:   "upokt",
  })

  // Validate the balance.
  require.NoError(s.T(), err)
  require.Equal(s.T(), fundAmount, balRes.GetBalance().Amount.Int64())
}

// Run the ExampleIntegrationSuite.
func TestExampleTestSuite(t *testing.T) {
    suite.Run(t, new(ExampleTestSuite))
}
```

## Implementing a Test Suite

// TODO_DOCUMENT(@bryanchriswhite)

### Test Suite Gotchas

- **Setup**: You MAY need to call `SetupXXX()`: check embedded suites for any required setup and copy-paste
- **Accessing Test State**: Avoid using `s.T()` in methods of suites which are intended to be embedded in other suites; pass a `*testing.T` argument instead.
- **Inheritance**: Inheriting multiple suites is hard since only one can be embedded anonymously: others will have to accessed via a named field.

// TODO_DOCUMENT(@bryanchriswhite): Add a `testutil/integration/suites.doc.go` with testable examples.
