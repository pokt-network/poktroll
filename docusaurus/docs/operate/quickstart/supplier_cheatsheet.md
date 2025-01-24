---
sidebar_position: 6
title: Supplier (RelayMiner) Cheat Sheet
---

## Supplier Cheat Sheet <!-- omit in toc -->

This guide provides quick reference commands for setting up a **Supplier** and
running a **RelayMiner** on Pocket Network.

For detailed instructions, troubleshooting, and observability setup, see the
[Supplier Walkthrough](./../run_a_node/supplier_walkthrough.md).

:::note

These instructions are intended to run on a Linux machine.

TODO_TECHDEBT(@olshansky): Adapt instructions to be macOS friendly in order to
streamline development and reduce friction for any new potential contributor.

:::

- [Pre-Requisites](#pre-requisites)
  - [Context](#context)
- [Account Setup](#account-setup)
  - [Create and fund the `Supplier` account](#create-and-fund-the-supplier-account)
  - [Prepare your environment](#prepare-your-environment)
- [Supplier Configuration](#supplier-configuration)
  - [Fund the Supplier account](#fund-the-supplier-account)
  - [Stake the Supplier](#stake-the-supplier)
- [RelayMiner Configuration](#relayminer-configuration)
  - [Configure the RelayMiner](#configure-the-relayminer)
  - [Start the RelayMiner](#start-the-relayminer)
  - [Secure vs Non-Secure `query_node_grpc_url`](#secure-vs-non-secure-query_node_grpc_url)
- [Supplier FAQ](#supplier-faq)
  - [What Supplier operations are available?](#what-supplier-operations-are-available)
  - [What Supplier queries are available?](#what-supplier-queries-are-available)
  - [How do I query for all existing onchain Suppliers?](#how-do-i-query-for-all-existing-onchain-suppliers)

## Pre-Requisites

1. Make sure to [install the `poktrolld` CLI](../user_guide/poktrolld_cli.md).
2. Make sure you know how to [create and fund a new account](../user_guide/create-new-wallet.md).
3. You have either [staked a new `service` or found an existing one](./service_cheatsheet.md).
4. `[Optional]` You can run things locally or have dedicated long-running hardware. See the [Docker Compose Cheat Sheet](./docker_compose_debian_cheatsheet#deploy-your-server) if you're interested in the latter.

:::warning

You can append `--keyring-backend test` to all the `poktrolld` commands throughout
this guide to avoid entering the password each time.

This is not recommended but provided for convenience for NON PRODUCTION USE ONLY.

⚠️ Use at your own risk. ⚠️

:::

### Context

This document is a cheat sheet to get you quickly started with two things:

1. Staking an onchain `Supplier`
2. Deploying an offchain `RelayMiner`

By the end of it, you should be able to serve Relays offchain, and claim onchain rewards.

## Account Setup

### Create and fund the `Supplier` account

Create a new key pair for the `Supplier`

```bash
poktrolld keys add supplier
```

### Prepare your environment

For convenience, we're setting several environment variables to streamline
the process of interacting with the Shannon network:

We recommend you put these in your `~/.bashrc` file:

```bash
export NODE="https://shannon-testnet-grove-rpc.beta.poktroll.com"
export NODE_FLAGS="--node=https://shannon-testnet-grove-rpc.beta.poktroll.com"
export TX_PARAM_FLAGS="--gas=auto --gas-prices=1upokt --gas-adjustment=1.5 --chain-id=pocket-beta --yes"
export SUPPLIER_ADDR=$(poktrolld keys show supplier -a)
```

:::tip

As an alternative to appending directly to `~/.bashrc`, you can put the above
in a special `~/.poktrollrc` and add `source ~/.poktrollrc` to
your `~/.profile` (or `~/.bashrc`) file for a cleaner organization.

:::

## Supplier Configuration

### Fund the Supplier account

Run the following command to get the `Supplier`:

```bash
echo "Supplier address: $SUPPLIER_ADDR"
```

Then use the [Shannon Beta TestNet faucet](https://faucet.beta.testnet.pokt.network/) to fund the (supplier owner address) account.
See [Non-Custodial Staking](https://dev.poktroll.com/operate/configs/supplier_staking_config#non-custodial-staking) for more information about supplier owner vs operator and non-custodial staking.

Afterwards, you can query the balance using the following command:

```bash
poktrolld query bank balances $SUPPLIER_ADDR $NODE_FLAGS
```

:::tip

You can find all the explorers, faucets and tools at the [tools page](../../explore/tools.md).

:::

### Stake the Supplier

:::info

For an in-depth look at how to stake a supplier, see the [Supplier configuration docs](./../configs/supplier_staking_config.md).

These instructions help you stake a supplier for a specific service (POKT Morse)
using a pre-configured RPC endpoint ([Liquify](https://liquify.com/) public RPC endpoint).

:::

Retrieve your external IP address:

```bash
EXTERNAL_IP=$(curl -4 ifconfig.me/ip)
```

Choose a port that'll be publicly accessible from the internet (e.g. `8545`)
and expose it.

You can use the following command for OSs that use `ufw` (learn more [here](https://wiki.archlinux.org/title/Uncomplicated_Firewall)):

```bash
sudo ufw allow 8545/tcp
```

Create a Supplier stake configuration file:

```bash
cat <<🚀 > /tmp/stake_supplier_config.yaml
owner_address: $SUPPLIER_ADDR
operator_address: $SUPPLIER_ADDR
stake_amount: 1000069upokt
default_rev_share_percent:
  $SUPPLIER_ADDR: 100
services:
  - service_id: "morse"
    endpoints:
      - publicly_exposed_url: http://$EXTERNAL_IP:8545
        rpc_type: JSON_RPC
🚀
```

And run the following command to stake the `Supplier`:

```bash
poktrolld tx supplier stake-supplier --config /tmp/stake_supplier_config.yaml --from=$SUPPLIER_ADDR $TX_PARAM_FLAGS $NODE_FLAGS
```

After about a minute, you can check the `Supplier`'s status like so:

```bash
poktrolld query supplier show-supplier $SUPPLIER_ADDR $NODE_FLAGS
```

## RelayMiner Configuration

### Configure the RelayMiner

```bash
cat <<🚀 > /tmp/relayminer_config.yaml
default_signing_key_names:
  - supplier
smt_store_path: /home/pocket/.poktroll/smt
pocket_node:
  query_node_rpc_url: https://shannon-testnet-grove-rpc.beta.poktroll.com
  query_node_grpc_url: https://shannon-testnet-grove-grpc.beta.poktroll.com:443
  tx_node_rpc_url: https://shannon-testnet-grove-rpc.beta.poktroll.com
suppliers:
  - service_id: "morse"
    service_config:
      backend_url: "https://pocket-rpc.liquify.com"
      publicly_exposed_endpoints:
        - $EXTERNAL_IP
    listen_url: http://0.0.0.0:8545
metrics:
  enabled: false
  addr: :9090
pprof:
  enabled: false
  addr: :6060
🚀
```

### Start the RelayMiner

```bash
poktrolld \
    relayminer \
    --grpc-insecure=false \
    --log_level=debug \
    --config=/tmp/relayminer_config.yaml
```

### Secure vs Non-Secure `query_node_grpc_url`

In `/tmp/relayminer_config.yaml`, you'll see that we specify an endpoint
for `query_node_grpc_url` which is TLS terminated.

If `grpc-insecure=true` then it **MUST** be an HTTP port, no TLS. Once you have
an endpoint exposed, it can be validated like so:

```bash
grpcurl -plaintext <host>:<port> list
```

<!--
TODO_TECHDEBT(@olshansk): Remove this comment.

Use at your own risk.

The Grove team temporarily exposed an unmaintained endpoint on one of our validators
for Beta Testnet at `http://149.28.34.68:9090`.

`grpcurl -plaintext 149.28.34.68:9090 list`
-->

If `grpc-insecure=false`, then it **MUST** be an HTTPS port, with TLS.

The Grove team exposed one such endpoint on one of our validators for Beta Testnet
at `https://shannon-testnet-grove-grpc.beta.poktroll.com:443`.

It can be validated with:

```bash
grpcurl shannon-testnet-grove-grpc.beta.poktroll.com:443 list
```

Note that no `-plaintext` flag is required when an endpoint is TLS terminated and
must be omitted if it is not.

:::tip

You can replace both `http` and `https` with `tcp` and it should work the same way.

:::

## Supplier FAQ

### What Supplier operations are available?

```bash
poktrolld tx supplier -h
```

### What Supplier queries are available?

```bash
poktrolld query supplier -h
```

### How do I query for all existing onchain Suppliers?

Then, you can query for all services like so:

```bash
poktrolld query supplier list-suppliers --node https://shannon-testnet-grove-rpc.beta.poktroll.com --output json | jq
```
