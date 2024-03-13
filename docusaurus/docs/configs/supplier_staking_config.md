---
title: Supplier staking config
sidebar_position: 3
---

# Supplier staking config <!-- omit in toc -->

_This document describes the configuration file used by the `Supplier` to submit
a stake transaction required to provide RPC services on the Pocket Network._

- [Usage](#usage)
- [Configuration](#configuration)
  - [`stake_amount`](#stake_amount)
  - [`services`](#services)
    - [`service_id`](#service_id)
    - [`endpoints`](#endpoints)
      - [`url`](#url)
      - [`rpc_type`](#rpc_type)
- [Example](#example)

## Usage

The `stake-supplier` transaction submission command accepts a `--config` flag
that points to a `yaml` configuration file that defines their staking
configuration. This includes, but is not limited to things like `stake_amount`, the `service`s, their respective advertised `endpoints`, etc.

The following is an example command of how to stake a supplier
in a LocalNet environment.

:::warning

TestNet is not ready as of writing this documentation, so you may
need to adjust the command below appropriately.

:::

```bash
poktrolld tx supplier stake-supplier \
  --home=./poktroll \
  --config ./stake_config.yaml \
  --keyring-backend test \
  --from supplier1 \
  --node tcp://poktroll-node:36657
```

## Configuration

### `stake_amount`

_`Required`_, _`Non-empty`_

```yaml
stake_amount: <number>upokt
```

Defines the amount of `upokt` to stake for the `Supplier` account.
This amount covers all the `service`s defined in the `services` section.

:::note

If the `Supplier` account already has a stake and wishes to change or add
to the `service`s that it provides, then it MUST to increase the current
`stake_amount` by at least `1upokt`.

For example, if the current stake is `1000upokt` and the `Supplier` wants to add a new `service` then `stake_amount: 1001upokt` should be specified in the configuration file. This will increase the stake by `1upokt` and deduct `1upokt` from the `Supplier`'s account balance.

:::

### `services`

_`Required`_, _`Non-empty`_

```yaml
services:
  - service_id: <string>
    endpoints:
      - url: <protocol>://<hostname>:<port>
        rpc_type: <string>
```

`services` define the list of services that the `Supplier` wants to provide.
It takes the form of a list of `service` objects. Each `service` object
consists of a `service_id` and a list of `endpoints` that the `Supplier` will
advertise on the Pocket Network.

#### `service_id`

_`Required`_

`service_id` is a string that uniquely identifies the service that the `Supplier`
is providing. It MUST 8 characters or less and composed of alphanumeric characters,
underscores, and dashes only.

For example, it must match the regex `^[a-zA-Z0-9_-]{1,8}$`, and spaces are disallowed.

#### `endpoints`

_`Required`_, _`Non-empty`_

`endpoints` is a list of `endpoint` objects that the `Supplier` will advertise
to the Pocket Network. Each `endpoint` object consists of an `url` and a `rpc_type`.

##### `url`

_`Required`_

The `url` defines the endpoint for sending `RelayRequests` from the Pocket Network's `Gateways` and `Applications`. Typically, this endpoint is provided by a RelayMiner, which acts as a proxy. This setup means that the specified URL directs to the RelayMiner's endpoint, which in turn, routes the requests to the designated service node.

- **Example**: `https://etherium-relayminer1.relayminers.com:443` demonstrates how a RelayMiner endpoint, typically offered by the RelayMiner, proxies requests to an Etherium backend data node.


##### `rpc_type`

_`Required`_

`rpc_type` is a string that defines the type of RPC service that the `Supplier`
is providing. The `rpc_type` MUST be one of the [supported types found here](https://github.com/pokt-network/poktroll/tree/main/pkg/relayer/config/types.go#L8)

## Example

A full example of the configuration file could be found at [supplier_staking_config.yaml](https://github.com/pokt-network/poktroll/tree/main/localnet/poktrolld/config/supplier1_stake_config.yaml)
