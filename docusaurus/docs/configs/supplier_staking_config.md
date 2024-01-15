---
title: Supplier staking config
sidebar_position: 3
---

# Supplier staking config

_This document describes the configuration file used by the `Supplier` to submit
a stake transaction required to provide RPC services on the Pocket Network._

- [Supplier staking config](#supplier-staking-config)
- [Usage](#usage)
- [Configuration](#configuration)
  - [`stake_amount`](#stake_amount)
  - [`services`](#services)
    - [`service_id`](#service_id)
    - [`endpoints`](#endpoints)
      - [`url`](#url)
      - [`rpc_type`](#rpc_type)
- [Example](#example)

# Usage

The `stake-supplier` transaction submission command accepts a `--config` flag
that points to a `yaml` configuration file that defines the `stake_amount`,
the `service`s and their respective advertised `endpoints`.

```bash
poktrolld tx supplier stake-supplier \
  --home=./poktroll \
  --config ./stake_config.yaml \
  --keyring-backend test \
  --from pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4 \
  --node tcp://poktroll-node:36657
```

# Configuration

The configuration file consists of a `stake_amount` entry denominated in `upokt`
and a `services` section that defines the list of services that the `Supplier`
wants to provide.

## `stake_amount`
_`Required`_

```yaml
stake_amount: <number>upokt
```

Defines the amount of `upokt` to stake for the `Supplier` account. This amount
covers all the `service`s defined in the `services` section.

_NOTE: If the `Supplier` account already has a stake and wishes to change or add
to the `service`s that it provides, then it needs to increase the current
`stake_amount` by at least `1upokt`.
(i.e. If the current stake is `1000upokt` and the `Supplier` wants to add a new
`service` then `stake_amount: 1001upokt` should be specified in the configuration
file; increasing it by `1upokt` and deducting `1upokt` from the `Supplier`'s
account balance.)_

## `services`
_`Required`_, _`Non-empty`_

```yaml
services:
  - service_id: <string>
    endpoints:
      - url: <protocol>://<hostname>:<port>
        rpc_type: <string>
```

`services` define the list of services that the `Supplier` wants to provide,
which takes the form of a list of `service` objects. Each `service` object
consists of a `service_id` and a list of `endpoints` that the `Supplier` will
advertise on the Pocket Network.

### `service_id`
_`Required`_

`service_id` is a string that uniquely identifies the service that the `Supplier`
is providing and MUST be of 8 characters or less allowing alphanumeric characters,
underscores, and dashes only (i.e. match the regex `^[a-zA-Z0-9_-]{1,8}$`, no spaces
allowed).

### `endpoints`
_`Required`_, _`Non-empty`_

`endpoints` is a list of `endpoint` objects that the `Supplier` will advertise
to the Pocket Network. Each `endpoint` object consists of an `url` and a `rpc_type`.

#### `url`
_`Required`_

`url` is a string formatted URL that defines the endpoint that MUST be reachable by
`Gateways` and `Applications` to send `RelayRequests` to.

#### `rpc_type`
_`Required`_

`rpc_type` is a string that defines the type of RPC service that the `Supplier`
is providing. The `rpc_type` MUST be one of the [supported types](https://github.com/pokt-network/poktroll/tree/main/pkg/relayer/config/types.go#L8)

# Example

A full example of the configuration file could be found at [supplier_staking_config.yaml](https://github.com/pokt-network/poktroll/tree/main/localnet/poktrolld/config/supplier1_stake_config.yaml)