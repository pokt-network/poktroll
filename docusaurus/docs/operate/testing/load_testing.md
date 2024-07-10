---
sidebar_position: 1
title: Load Testing
---

# Load Testing <!-- omit in toc -->

Poktroll load-testing suite.

- [Overview](#overview)
- [Dependencies](#dependencies)
- [Load Test Manifests](#load-test-manifests)
- [Test Features](#test-features)
- [Executing Tests](#executing-tests)
  - [LocalNet Environment](#localnet-environment)
    - [Interpreting Results](#interpreting-results)
  - [Non-Ephemeral Networks (TestNets, MainNet, etc)](#non-ephemeral-networks-testnets-mainnet-etc)
    - [Prerequisites](#prerequisites)
    - [Manifest Modification](#manifest-modification)
    - [Test Execution](#test-execution)
    - [Result Analysis](#result-analysis)
- [Developing Custom Tests](#developing-custom-tests)

## Overview

The load-testing suite is built on [Gherkin](https://cucumber.io/docs/gherkin/), enabling the creation of simple and human-readable tests.

## Dependencies

- [LocalNet](../infrastructure/localnet.md) (for local suite execution)
- [Golang](https://go.dev/dl/)

## Load Test Manifests

Load test manifests are YAML files that define the network environment for test execution. These files specify properties such as blockchain account addresses for funding and staking applications, and for utilization by suppliers and gateways. The [LocalNet manifest](https://github.com/pokt-network/poktroll/blob/main/load-testing/loadtest_manifest_localnet.yaml) serves as a comprehensive example, including detailed comments for each manifest property.

## Test Features

Test features are located in the [load-testing/tests](https://github.com/pokt-network/poktroll/tree/main/load-testing/tests) directory, encompassing various scenarios.

Feature files are composed of one or more [`Scenario`](https://cucumber.io/docs/gherkin/reference/?sbsearch=Scenarios)s (or [`Scenario Outline`](https://cucumber.io/docs/gherkin/reference/?sbsearch=Scenarios#scenario-outline)s), each of which is composed of one or more [steps](https://cucumber.io/docs/gherkin/reference/#steps). For example:

```gherkin
Feature: Loading anvil node only
  Scenario Outline: Anvil can handle the maximum number of concurrent users
    Given anvil is running
    And load of <num_requests> concurrent requests for the "eth_blockNumber" JSON-RPC method
    Then load is handled within <timeout> seconds

    Examples:
      | num_requests | timeout |
      | 10           | 1       |
      | 100          | 1       |
      | 1000         | 5       |
      | 10000        | 10      |
```

This natural language syntax is parsed and used to match and execute the corresponding [step definitions](https://cucumber.io/docs/cucumber/step-definitions/?lang=javascript) using [gocuke](https://github.com/regen-network/gocuke).

## Executing Tests

### LocalNet Environment

To execute tests on LocalNet:

1. Ensure [LocalNet](../infrastructure/localnet.md) is operational.
2. In the `localnet_config.yaml` file, set `gateways.count` and `relayminers.count` to `3`.
3. Run `make acc_initialize_pubkeys` to initialize blockchain state public keys.
4. Run `make test_load_relays_stress_localnet` to run the LocalNet stress-test.

#### Interpreting Results

- The CLI output displays standard Go test results. Successful tests are indicated by `PASS`, while failures are denoted by `FAIL` with accompanying error messages.
- During test execution, the observability stack continuously collects metric data from off-chain actors. On LocalNet, [Grafana is accessible on port 3003](http://localhost:3003/?orgId=1). The
  [Stress test](http://localhost:3003/d/ddkakqetrti4gb/protocol-stress-test?orgId=1&refresh=5s)
  and [Load Testing](http://localhost:3003/d/fdjwb9u9t9ts0e/protocol-load-testing?orgId=1) dashboards provide valuable
  insights into system status.

### Non-Ephemeral Networks (TestNets, MainNet, etc)

These networks are generated with random addresses, necessitating modifications to the load test manifest to reflect network-specific accounts.

:::info
Note: Such networks typically involve other participants, allowing load testing against off-chain actors deployed by third parties. Consequently, metrics and logs may not be available when testing against uncontrolled software. For comprehensive observability, consider creating a new service with custom gateways and suppliers, and conduct tests against this controlled environment.
:::

#### Prerequisites

- A account with sufficient tokens for application funding and staking, accessible on the local keychain (e.g., `poktrolld keys list`).
- A list of target gateways to which relay requests will be sent throughout the course of the test.
- For custom service testing, ensure supplier(s') corresponding relayminer(s) and custom service process are properly configured, running, and ready to process requests.

#### Manifest Modification

Using [loadtest_manifest_example.yaml](https://github.com/pokt-network/poktroll/blob/main/load-testing/loadtest_manifest_example.yaml) as a template, modify the values to align with the test requirements.

#### Test Execution

Utilize the provided makefile target to run the `relays_stress.feature` with the modified manifest. By default, the example manifest file is used. You can specify a different manifest file by setting the `LOAD_TEST_CUSTOM_MANIFEST` environment variable.

To run the stress test using the default manifest:

```bash
make test_load_relays_stress_custom
```

To run the stress test with a custom manifest:

```bash
LOAD_TEST_CUSTOM_MANIFEST=your_new_manifest.yaml make test_load_relays_stress_custom
```

#### Result Analysis

For tests conducted against community-hosted suppliers and gateways, analysis is limited to blockchain transactions and test output. When testing against a custom-deployed service, comprehensive observability is available, including metrics, logs, and system behavior under load, accessible through Grafana or other configured monitoring tools.

## Developing Custom Tests

For custom test development, refer to the [gocuke documentation](https://github.com/regen-network/gocuke?tab=readme-ov-file#quick-start). The [anvil test](https://github.com/pokt-network/poktroll/blob/main/load-testing/tests/anvil_test.go) provides a small but practical reference implementation.