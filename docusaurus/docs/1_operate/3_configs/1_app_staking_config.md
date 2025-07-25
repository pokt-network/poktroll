---
title: Application staking config
sidebar_position: 1
---

This document describes the configuration file used by the `Application` actor
to submit a `stake` transaction, **which is a prerequisite** for it to use the
Pocket Network's RPC services.

:::tip

You can find a fully featured example configuration at [application1_stake_config.yaml](https://github.com/pokt-network/poktroll/blob/main/localnet/pocketd/config/application1_stake_config.yaml).

:::

- [Gov Param References \& Values](#gov-param-references--values)
- [Usage](#usage)
- [Configuration](#configuration)
  - [`stake_amount`](#stake_amount)
  - [`service_ids`](#service_ids)

## Gov Param References & Values

- Application module governance params can be found [here](../../3_protocol/governance/2_gov_params.md).
- Application module Beta parameter values can be found [here](https://github.com/pokt-network/poktroll/blob/main/tools/scripts/params/bulk_params_beta/application_params.json).
- Application module Main parameter values can be found [here](https://github.com/pokt-network/poktroll/blob/main/tools/scripts/params/bulk_params_main/application_params.json).

## Usage

The `stake-application` transaction submission command accepts a `--config` flag
that points to a `yaml` configuration file that defines the `stake_amount` and
`service_ids` which the `Application` is allowed to use.

```bash
pocketd tx application stake-application \
  --home=./pocket \
  --config ./stake_config.yaml \
  --keyring-backend test \
  --from application1 \
  --network=<network> #e.g. local, alpha, beta, main
```

## Configuration

The configuration file consists of a `stake_amount` entry denominated in `upokt`
and a `service_ids` list defining the services the `Application` is interested
in consuming.

### `stake_amount`

_`Required`_

```yaml
stake_amount: <number>upokt
```

Defines the amount of `upokt` to stake from the `Application` to be able to
consume the services. This amount will be transferred from the Application's
account balance and locked. It will be deducted at the end of every session
based on the Application's usage.

### `service_ids`

_`Required`_, _`Non-empty`_

```yaml
service_ids:
  - <string>
```

:::warning

The `service_ids` list must contain a unique entry.

The current protocol requires the `service_ids` list to contain **EXACTLY ONE** entry
to prevent `Application`s from over-servicing.

A detailed explanation of why this is the case can be found in
[Tokenomics/TLM](../../3_protocol/tokenomics/4_token_logic_modules_mint_equal_burn.md).

:::

Defines the list of services the `Application` is willing to consume on the
Pocket network. Each entry in the list is a `service_id` that identifies a service
that is available on Pocket network.

It MUST be a string of 8 or fewer alphanumeric characters, underscores, and
dashes (i.e. matching the regex `^[a-zA-Z0-9_-]{1,8}$`).
