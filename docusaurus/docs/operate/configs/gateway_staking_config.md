---
title: Gateway staking config
sidebar_position: 2
---

# Gateway staking config <!-- omit in toc -->

This document describes the configuration file used by the `Gateway` actor
to submit a `stake` transaction, **which is a prerequisite** for it proxy relays
on behalf of `Application`s.

:::tip

You can find a fully featured example configuration at [gateway1_stake_config.yaml](https://github.com/pokt-network/poktroll/tree/main/localnet/poktrolld/config/gateway1_stake_config.yaml).

:::

- [Usage](#usage)
- [Configuration](#configuration)
  - [`stake_amount`](#stake_amount)

## Usage

The `stake-gateway` transaction submission command accepts a `--config` flag
that points to a `yaml` configuration file that defines the `stake_amount` the
`Gateway` is willing to lock.

:::warning

TestNet is not ready as of writing this documentation, so you may
need to adjust the command below appropriately.

:::

```bash
poktrolld tx gateway stake-gateway \
  --home=./poktroll \
  --config ./stake_config.yaml \
  --keyring-backend test \
  --from gateway1 \
  --node tcp://poktroll-node:26657
```

## Configuration

The configuration file consists of the `stake_amount` entry denominated in `upokt`.

### `stake_amount`

_`Required`_

```yaml
stake_amount: <number>upokt
```

Defines the amount of `upokt` to stake by the `Gateway` to be able to serve
`RelayRequest` on the Pocket network on behalf of `Application`s.
