---
sidebar_position: 2
title: Load Testing - DevNet
---

# Load Testing on DevNets <!-- omit in toc -->

A guide on how to perform load testing on DevNets.

- [Overview](#overview)
- [Prerequisites](#prerequisites)
  - [1. Create and configure the DevNet](#1-create-and-configure-the-devnet)
  - [2. Stake the necessary actors](#2-stake-the-necessary-actors)
  - [3. Configure the load test manifest](#3-configure-the-load-test-manifest)
- [Full example](#full-example)

## Overview

We can create DevNets that are suitable for running load tests.

:::warning
DevNets created with GitHub PRs using `devnet-test-e2e` tags are not suitable for load testing, as they only provision a
single instance of each offchain actor. We can create custom DevNets with multiple instances of each offchain actor for load testing purposes.
:::

## Prerequisites

### 1. Create and configure the DevNet

Please refer to the DevNet creation guide [here](../infrastructure/devnet.md#how-to-create).

### 2. Stake the necessary actors

- Depending on your load testing requirements, you may need to stake one or more `gateways` and `suppliers`.
- [DevNet documentation](../infrastructure/devnet.md#stake-actors) provides more details about staking actors in DevNets.

### 3. Configure the load test manifest

[Load Testing documentation](./load_testing.md#manifest-modification) provides information on how the load test manifest
can be modified to run against DevNets. DevNets are not much different from TestNets, but they **do not** have randomly
generated accounts. Instead, the accounts are derived from the ignite `config.yaml` (as in LocalNet) for convenience.

## Full example

1. DevNet `sophon` can be created with the following devnet-config file:

```yaml
networkName: "sophon"

image:
  tag: sha-7042be3

path_gateways:
  count: 3
relayminers:
  count: 3
```

2. Gateways and suppliers can be staked using the following commands:

:::info
`supplier1` and `gateway1` are pre-staked as part of the genesis generation process.
:::

```bash
POCKET_NODE=https://devnet-sophon-validator-rpc.poktroll.com make gateway2_stake
POCKET_NODE=https://devnet-sophon-validator-rpc.poktroll.com make gateway3_stake
POCKET_NODE=https://devnet-sophon-validator-rpc.poktroll.com make supplier2_stake
POCKET_NODE=https://devnet-sophon-validator-rpc.poktroll.com make supplier3_stake
```

3. Update the manifest. The content of
[loadtest_manifest_example.yaml](https://github.com/pokt-network/poktroll/blob/main/load-testing/loadtest_manifest_example.yaml)
can be modified as follows:

```yaml
# This file is used to configure the load test for non-ephemeral chains.
# It is intended to target a remote environment, such as a devnet or testnet.
is_ephemeral_chain: false

# testnet_node is the URL of the node that the load test will use to query the
# chain and submit transactions.
testnet_node: https://devnet-sophon-validator-rpc.poktroll.com

# The service ID to request relays from.
service_id: "anvil"

# The address of the account that will be used to fund the application accounts
# so that they can stake on the network.
# TODO_TECHDEBT(@bryanchriswhite, #512): Replace with faucet address.
funding_account_address: pokt1awtlw5sjmw2f5lgj8ekdkaqezphgz88rdk93sk # address for faucet account

# In non-ephemeral chains, the gateways are identified by their addresses.
gateways:
  - address: pokt15vzxjqklzjtlz7lahe8z2dfe9nm5vxwwmscne4
    exposed_url: https://devnet-sophon-gateway-1.poktroll.com
  - address: pokt15w3fhfyc0lttv7r585e2ncpf6t2kl9uh8rsnyz
    exposed_url: https://devnet-sophon-gateway-2.poktroll.com
  - address: pokt1zhmkkd0rh788mc9prfq0m2h88t9ge0j83gnxya 
    exposed_url: https://devnet-sophon-gateway-3.poktroll.com
```

4. Run the test:

```bash
make test_load_relays_stress_example
```

5. Observe the results:

You can see the performance of the requests on [Grafana dashboards](https://grafana.poktroll.com/d/nginx/nginx-ingress-controller).
The DevNets have LoadBalancers which allow for more metrics about network load and latency. When looking
at the `NGINX Ingress controller` Dashboard, make sure to change the namespace to match the DevNet name.