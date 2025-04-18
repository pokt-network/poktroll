---
sidebar_position: 6
title: App & PATH Gateway (~30 min)
---

## App & PATH Gateway Cheat Sheet <!-- omit in toc -->

This guide provides quick reference commands for setting up and running a **Gateway**
on Pocket Network.

For detailed instructions, troubleshooting, and observability setup, see the
[Gateway Walkthrough](../walkthroughs/gateway_walkthrough.md).

:::note

These instructions are intended to run on a Linux machine.

TODO_TECHDEBT(@olshansky): Adapt instructions to be macOS friendly in order to
streamline development and reduce friction for any new potential contributor.

:::

- [Pre-Requisites](#pre-requisites)
- [Account Setup](#account-setup)
  - [Create and fund the `Gateway` and `Application` accounts](#create-and-fund-the-gateway-and-application-accounts)
  - [Prepare your environment](#prepare-your-environment)
  - [Fund the Gateway and Application accounts](#fund-the-gateway-and-application-accounts)
- [Gateway and Application Configurations](#gateway-and-application-configurations)
  - [Stake the `Gateway`](#stake-the-gateway)
  - [Stake the delegating `Application`](#stake-the-delegating-application)
  - [Delegate the `Application` to the `Gateway`](#delegate-the-application-to-the-gateway)
- [`PATH` Gateway Setup](#path-gateway-setup)

## Pre-Requisites

1. Make sure to [install the `pocketd` CLI](../../tools/user_guide/pocketd_cli.md).
2. Make sure you know how to [create and fund a new account](../../tools/user_guide/create-new-wallet.md).

:::warning

You can append `--keyring-backend test` to all the `pocketd` commands throughout
this guide to avoid entering the password each time.

This is not recommended but provided for convenience for NON PRODUCTION USE ONLY.

⚠️ Use at your own risk. ⚠️

:::

## Account Setup

### Create and fund the `Gateway` and `Application` accounts

Create a new key pair for the delegating `Application`:

```bash
pocketd keys add application
```

Create a new key pair for the `Gateway`:

```bash
pocketd keys add gateway
```

### Prepare your environment

For convenience, we're setting several environment variables to streamline
the process of interacting with the Shannon network:

We recommend you put these in your `~/.bashrc` file:

```bash
export POCKET_NODE="https://shannon-testnet-grove-rpc.beta.poktroll.com"
export NODE_FLAGS="--node=$POCKET_NODE"
export TX_PARAM_FLAGS="--gas=auto --gas-prices=1upokt --gas-adjustment=1.5 --chain-id=pocket-beta --yes"
export GATEWAY_ADDR=$(pocketd keys show gateway -a)
export APP_ADDR=$(pocketd keys show application -a)
```

:::tip

As an alternative to appending directly to `~/.bashrc`, you can put the above
in a special `~/.pocketrc` and add `source ~/.pocketrc` to
your `~/.profile` (or `~/.bashrc`) file for a cleaner organization.

:::

### Fund the Gateway and Application accounts

Run the following command to get the `Gateway` and `Application` addresses:

```bash
echo "Gateway address: $GATEWAY_ADDR"
echo "Application address: $APP_ADDR"
```

Then use the [Shannon Beta TestNet faucet](https://faucet.beta.testnet.pokt.network/) to fund the `Gateway`
and `Application` accounts respectively.

Afterwards, you can query their balances using the following command:

```bash
pocketd query bank balances $GATEWAY_ADDR $NODE_FLAGS
pocketd query bank balances $APP_ADDR $NODE_FLAGS
```

:::tip

You can find all the explorers, faucets and tools at the [tools page](../../tools/tools/source_code.md).

:::

## Gateway and Application Configurations

### Stake the `Gateway`

Create a Gateway stake configuration file:

```bash
cat <<🚀 > /tmp/stake_gateway_config.yaml
stake_amount: 1000000upokt
🚀
```

And run the following command to stake the `Gateway`:

```bash
pocketd tx gateway stake-gateway --config=/tmp/stake_gateway_config.yaml --from=$GATEWAY_ADDR $TX_PARAM_FLAGS $NODE_FLAGS
```

After about a minute, you can check the `Gateway`'s status like so:

```bash
pocketd query gateway show-gateway $GATEWAY_ADDR $NODE_FLAGS
```

### Stake the delegating `Application`

Create an Application stake configuration file:

```bash
cat <<🚀 > /tmp/stake_app_config.yaml
stake_amount: 100000000upokt
service_ids:
  - "anvil"
🚀
```

And run the following command to stake the `Application`:

```bash
pocketd tx application stake-application --config=/tmp/stake_app_config.yaml --from=$APP_ADDR $TX_PARAM_FLAGS $NODE_FLAGS
```

After about a minute, you can check the `Application`'s status like so:

```bash
pocketd query application show-application $APP_ADDR $NODE_FLAGS
```

### Delegate the `Application` to the `Gateway`

```bash
pocketd tx application delegate-to-gateway $GATEWAY_ADDR --from=$APP_ADDR $TX_PARAM_FLAGS $NODE_FLAGS
```

After about a minute, you can check the `Application`'s status like so:

```bash
pocketd query application show-application $APP_ADDR $NODE_FLAGS
```

## `PATH` Gateway Setup

:::tip

For instructions on setting up a `PATH` Gateway, see the [Configure PATH for Shannon](https://path.grove.city/develop/path/cheatsheet_shannon#2-configure-path-for-shannon) sections of the `PATH` documentation.

:::
