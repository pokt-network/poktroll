---
sidebar_position: 1
title: Load Testing
---

# 🚀 Load Testing

Pocket Network's comprehensive load testing suite for evaluating network performance and reliability under various stress conditions.

- [🚀 Load Testing](#-load-testing)
  - [Overview](#overview)
  - [⚡ Quick Start](#-quick-start)
  - [🧪 Test Types](#-test-types)
    - [🎯 Default Load Test (Single Supplier)](#-default-load-test-single-supplier)
    - [🌐 Multi-Supplier Load Test](#-multi-supplier-load-test)
    - [🔄 Claim Settlement Stability Test](#-claim-settlement-stability-test)
  - [📋 Test Manifests](#-test-manifests)
  - [🏃 Running Tests](#-running-tests)
    - [⚠️ Prerequisites](#️-prerequisites)
    - [🔧 LocalNet Setup](#-localnet-setup)
    - [🌍 Custom Network Testing](#-custom-network-testing)
  - [📊 Monitoring Results](#-monitoring-results)
    - [🖥️ Test Output](#️-test-output)
    - [📈 Grafana Dashboards (LocalNet)](#-grafana-dashboards-localnet)
  - [📝 Test Features Example](#-test-features-example)
  - [🎯 Available Commands](#-available-commands)

## Overview

The load-testing suite uses [Gherkin](https://cucumber.io/docs/gherkin/) for creating human-readable test scenarios that validate network behavior under load. Tests are located in the `load-testing/` directory and use YAML manifests to define network configurations.

## ⚡ Quick Start

:::tip Get Started in 2 Steps
The simplest way to run load tests is with the default LocalNet single supplier configuration:

1. Ensure [LocalNet](../networks/localnet.md) is running
2. Run the default load test:
   ```bash
   make test_load_relays_stress_localnet_single_supplier
   ```
:::

## 🧪 Test Types

### 🎯 Default Load Test (Single Supplier)

:::note Recommended Starting Point
The default test gradually increases the number of applications while maintaining a single supplier and gateway. This tests the supplier's ability to handle increasing load.
:::

**Configuration:**
- Initial: 4 applications, 1 gateway, 1 supplier
- Scaling: Adds 4 applications every 10 blocks up to 12 total
- Rate: 1 relay request per second per application

**Command:**
```bash
make test_load_relays_stress_localnet_single_supplier
```

### 🌐 Multi-Supplier Load Test

Tests the network with multiple suppliers and gateways scaling together.

**Configuration:**
- Initial: 4 applications, 1 gateway, 1 supplier
- Scaling: All actors scale together up to 3 suppliers and 3 gateways
- Rate: 1 relay request per second per application

**Command:**
```bash
make test_load_relays_stress_localnet
```

### 🔄 Claim Settlement Stability Test

This test maintains high constant load to validate that the network remains stable during claim settlement periods.

:::info What This Test Validates
This test checks the **session lifetime caching optimization**, which prevents `Relay`
serving freezes by:

- **Smart Cache Timing**: Instead of clearing cache every block, the system now clears
cache only at the end of each session
- **Reduced Node Pressure**: This prevents unnecessary requests to full nodes during
busy claim settlement blocks

The goal is to ensure this caching improvement prevents relay failures or timeouts during critical claim settlement periods.
:::

**Configuration:**
- Constant load: 50 applications, 50 gateways, 50 suppliers (no scaling)
- Rate: 2 relay requests per second per application
- Focus: Validate that no timeouts occur and relay success rates remain stable during claim settlement

**Key Validation Points:**
- No timeout spikes during claim settlement blocks
- Consistent relay success rates despite of full node unresponsiveness

:::warning High Resource Usage
This test uses significantly more resources than other tests due to the high number of concurrent actors and relay rate.
:::

## 📋 Test Manifests

Load test manifests are YAML files that define network configurations for different testing scenarios.

:::tip Configuration Files
Available manifest files:

- `loadtest_manifest_localnet_single_supplier.yaml` - Default single supplier configuration
- `loadtest_manifest_localnet.yaml` - Multi-supplier configuration
- `loadtest_manifest_example.yaml` - Template for custom networks
:::

## 🏃 Running Tests

### ⚠️ Prerequisites

Before running load tests, ensure your environment is properly configured:

:::danger Important Setup Steps
- [LocalNet](../networks/localnet.md) must be running
- Run `make acc_initialize_pubkeys` to initialize blockchain public keys
:::

### 🔧 LocalNet Setup

Proper LocalNet configuration is essential for successful load testing.

:::note Configuration Required
1. Configure `localnet_config.yaml`:
   - Set `gateways.count` and `relayminers.count` to match your test requirements
   - For single supplier tests: `relayminers.count = 1`
   - For multi-supplier tests: `relayminers.count = 3`

2. Start LocalNet and run tests using the commands above
:::

### 🌍 Custom Network Testing

You can run load tests against external networks like testnets or custom deployments by modifying the test manifest.

:::caution Advanced Usage
For testing against testnets or custom deployments:

1. Copy `loadtest_manifest_example.yaml` and modify it for your network
2. Update account addresses, RPC endpoints, and actor configurations
3. Run with custom manifest:
   ```bash
   LOAD_TEST_CUSTOM_MANIFEST=your_manifest.yaml make test_load_relays_stress_custom
   ```
:::

## 📊 Monitoring Results

### 🖥️ Test Output

Load tests provide detailed output about test execution and results.

:::info Result Interpretation
- `PASS` indicates successful test completion
- `FAIL` shows failures with error details
- Monitor relay success/failure rates in the output
:::

### 📈 Grafana Dashboards (LocalNet)

LocalNet provides comprehensive observability through Grafana dashboards for real-time monitoring during load tests.

:::tip Real-time Monitoring
When running on LocalNet, access observability at [http://localhost:3003](http://localhost:3003):

- [Stress Test Dashboard](http://localhost:3003/d/ddkakqetrti4gb/protocol-stress-test?orgId=1&refresh=5s) - Real-time load metrics
- [Load Testing Dashboard](http://localhost:3003/d/fdjwb9u9t9ts0e/protocol-load-testing?orgId=1) - Comprehensive performance insights
:::

## 📝 Test Features Example

Load tests are written using Gherkin syntax, making them human-readable and easy to understand.

```gherkin
Feature: Loading gateway server with relays

  Scenario: Incrementing the number of relays and actors
    Given localnet is running
    And a rate of "1" relay requests per second is sent per application
    And the following initial actors are staked:
      | actor       | count |
      | application | 4     |
      | gateway     | 1     |
      | supplier    | 1     |
    And more actors are staked as follows:
      | actor       | actor inc amount | blocks per inc | max actors |
      | application | 4                | 10             | 12         |
      | gateway     | 1                | 10             | 1          |
      | supplier    | 1                | 10             | 1          |
    When a load of concurrent relay requests are sent from the applications
    Then the number of failed relay requests is "0"
```

## 🎯 Available Commands

Here are all the available make commands for running different types of load tests.

| Command | Purpose |
|---------|---------|
| `make test_load_relays_stress_localnet_single_supplier` | **Default** single supplier load test |
| `make test_load_relays_stress_localnet` | Multi-supplier load test |
| `make test_load_relays_stress_custom` | Custom manifest load test |