---
sidebar_position: 1
title: Load Testing
---

# Load Testing <!-- omit in toc -->

Poktroll load-testing suite.

- [Overview](#overview)
- [Dependencies](#dependencies)
- [Load Test manifests](#load-test-manifests)
- [Test Features](#test-features)
- [How to run tests](#how-to-run-tests)
  - [LocalNet](#localnet)
    - [Reading the results](#reading-the-results)
  - [Non-ephemeral networks (TestNets, MainNet, etc)](#non-ephemeral-networks-testnets-mainnet-etc)
    - [Prerequisites](#prerequisites)
    - [Modify the load test manifest](#modify-the-load-test-manifest)
    - [Run the test](#run-the-test)
    - [Reading the results](#reading-the-results-1)
- [How to write your own tests](#how-to-write-your-own-tests)

## Overview

We built a load-testing suite on top of [Gherkin](https://cucumber.io/docs/gherkin/) which allows to write simple and human readable tests.

## Dependencies

- (For local suite execution) [LocalNet](../infrastructure/localnet.md)
- [Golang](https://go.dev/dl/)

## Load Test manifests

Load test manifests are YAML files that describe the environment of the network the test can be run against. Properties 
such as what blockchain address to use to fund and stake applications, what suppliers and gateways to use are 
all covered in the manifest YAML file. [The LocalNet's manifest](https://github.com/pokt-network/poktroll/blob/main/load-testing/loadtest_manifest_localnet.yaml)
can be used as an example - it includes comments for each property in the manefest.

## Test Features

Test features are stored in the [load-testing/tests](https://github.com/pokt-network/poktroll/tree/main/load-testing/tests) directory,
covering various use cases.

As the load-testing suite is built on top of Gherkin, the features files contain human-readable load tests.
For example, here is a simple feature that checks if the `anvil` node can handle the maximum number of concurrent users:

```
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

The natural human-readable text is parsed by [gocuke](https://github.com/regen-network/gocuke). 

## How to run tests

### LocalNet

We have a handy make target to run the tests on the LocalNet.

1. Make sure [LocalNet](../infrastructure/localnet.md) is up and running;
2. In your `localnet_config.yaml` file, make sure to set `gateways.count` and `relayminers.count` to `3`;
3. Run `make acc_initialize_pubkeys` to initialize the public keys in the blockchain state;
4. Run `make test_load_relays_stress_localnet` to run all tests on LocalNet.

#### Reading the results

- The CLI output shows standard Go test output. If there are no issues during the test execution, you'll see `PASS`, otherwise the test will show `FAIL` and the error that caused the test to fail will be shown.
- As the test progresses, the obserability stack continously gathers the metric data from off-chain actors. On LocalNet, [Grafana can be accessed on 3003 port](http://localhost:3003/?orgId=1). `Stress test` and `Load Testing` dashboards can be helpful to understand current state of the system.

### Non-ephemeral networks (TestNets, MainNet, etc)

Such networks have been generated with random addresses, so we need to modify the load test manifest to reflect 
the accounts from that network.

::: info
Such networks usually have other participants and load testing can be performed against the off-chain actors deployed by
other people. As a result of running the test against the software you don't control and can't observe - you won't
get the metrics and logs. If you wish to gather metrics, logs and look at behavior of the off-chain actors, you can 
create a new service and deploy your own gateways and suppliers, and run the test against that new service. As a result
you'll get full observability information from the software you deployed.
:::

#### Prerequisites

- An address with tokens that will be used to fund and stake applications. It must be available on the local keychain (e.g. `poktrolld keys list`)
- A list of gateways to issue requests to. They could be gateways hosted by other people, or they can be your own gateways.
- If you are running a test on a custom service, then make sure the suppliers are set up and ready to accept requests.

#### Modify the load test manifest

Using [loadtest_manifest_example.yaml](https://github.com/pokt-network/poktroll/blob/main/load-testing/loadtest_manifest_example.yaml)
as a reference, modify it to reflect the values for your test.

#### Run the test

We have a handy makefile target to run the `relays_stress.feature` with the modified manifest (`loadtest_manifest_example.yaml`):

```bash
make test_load_relays_stress_example
```

#### Reading the results

If the test was ran against the suppliers and gateways that are not hosted by you, but rather the community members, you can
only look at the transactions on blockchain and the test output. If you deployed your own service, then you will have full observability 
and you should be able to see all the metrics, logs and behavior of the system under load in Grafana or any other monitoring tool that is set up for your service.

## How to write your own tests

Please refer to the [gocuke documentation](https://github.com/regen-network/gocuke?tab=readme-ov-file#quick-start). You
can also use a simple [anvil test](https://github.com/pokt-network/poktroll/blob/main/load-testing/tests/anvil_test.go) as a reference.