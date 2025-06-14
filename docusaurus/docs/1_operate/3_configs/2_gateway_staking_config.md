---
title: Gateway staking config
sidebar_position: 2
---

This document describes the configuration file used by the `Gateway` actor
to submit a `stake` transaction, **which is a prerequisite** for it proxy relays
on behalf of `Application`s.

:::tip

You can find a fully featured example configuration at [gateway1_stake_config.yaml](https://github.com/pokt-network/poktroll/tree/main/localnet/pocketd/config/gateway1_stake_config.yaml).

:::

- [Gov Param References \& Values](#gov-param-references--values)
- [Usage](#usage)
- [Configuration](#configuration)
  - [`stake_amount`](#stake_amount)

## Gov Param References & Values

- Gateway module governance params can be found [here](../../3_protocol/governance/2_gov_params.md).
- Gateway module Beta parameter values can be found [here](https://github.com/pokt-network/poktroll/blob/main/tools/scripts/params/bulk_params_beta/gateway_params.json).
- Gateway module Main parameter values can be found [here](https://github.com/pokt-network/poktroll/blob/main/tools/scripts/params/bulk_params_main/gateway_params.json).

## Usage

The `stake-gateway` transaction submission command accepts a `--config` flag
that points to a `yaml` configuration file that defines the `stake_amount` the
`Gateway` is willing to lock.

```bash
pocketd tx gateway stake-gateway \
  --home=./pocket \
  --config ./stake_config.yaml \
  --keyring-backend test \
  --from gateway1 \
  --network=<network> #e.g. local, alpha, beta, main
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
