---
sidebar_position: 6
title: Supplier (RelayMiner) Cheat Sheet
---

## Supplier Cheat Sheet <!-- omit in toc -->

This guide provides quick reference commands for setting up a **Supplier** and
running a **RelayMiner** on Pocket Network.

:::warning

These instructions are intended to run on a Linux machine.

TODO_TECHDEBT(@olshansky): Adapt the instructions to be macOS friendly.

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
  - [What Supplier transactions are available?](#what-supplier-transactions-are-available)
  - [What Supplier queries are available?](#what-supplier-queries-are-available)
  - [How do I query for all existing onchain Suppliers?](#how-do-i-query-for-all-existing-onchain-suppliers)

:::note

For detailed instructions, troubleshooting, and observability setup, see the [Supplier Walkthrough](./../run_a_node/supplier_walkthrough.md).

:::

## Pre-Requisites

1. Make sure to [install the `poktrolld` CLI](../user_guide/install.md).
2. Make sure you know how to [create and fund a new account](../user_guide/create-new-wallet.md).
3. You have either [staked a new `service` or found an existing one](./service_cheatsheet.md).
4. `[Optional]` You can run things locally or have dedicated long-running hardware. See the [Docker Compose Cheat Sheet](./docker_compose_debian_cheatsheet#deploy-your-server) if you're interested in the latter.

### Context

This document is a cheat sheet to get you quickly started with two things:

1. Staking an onchain `Supplier`
2. Deploying an offchain `RelayMiner`

By the end of it, you should be able to serve Relays onchain.

## Account Setup

### Create and fund the `Supplier` account

Create a new key pair for the `Supplier`

```bash
poktrolld keys add supplier

# Optionally, to avoid entering the password each time:
# poktrolld keys add supplier --keyring-backend test
```

:::tip

You can set the `--keyring-backend` flag to `test` to avoid entering the password
each time.

Learn more about [cosmos keyring backends here](https://docs.cosmos.network/v0.46/run-node/keyring.html).

:::

### Prepare your environment

For convenience, we're setting several environment variables to streamline
the process of interacting with the Shannon network:

We recommend you put these in your `~/.bashrc` file:

```bash
export NODE="https://shannon-testnet-grove-rpc.beta.poktroll.com"
export NODE_FLAGS="--node=https://shannon-testnet-grove-rpc.beta.poktroll.com"
export TX_PARAM_FLAGS="--gas=auto --gas-prices=1upokt --gas-adjustment=1.5 --chain-id=pocket-beta --yes"
export SUPPLIER_ADDR=$(poktrolld keys show supplier -a)

# Optionally, to avoid entering the password each time:
# export SUPPLIER_ADDR=$(poktrolld keys show supplier -a --keyring-backend test
```

:::tip

As an alternative to appending directly to `~/.bashrc`, you can put the above in a special `~/.poktrollrc` and add `source ~/.poktrollrc` to
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

The example below is a very quick and simple way to get you started by staking for
Pocket Network's Morse service on Shannon using a public RPC endpoint provided by
[Liquify](https://liquify.com/).

:::

Retrieve your external IP address:

```bash
EXTERNAL_IP=$(curl -4 ifconfig.me/ip)
```

Choose a port that'll be publicly accessible from the internet (e.g. `8545`)

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

# Optionally, to avoid entering the password each time:
# poktrolld tx supplier stake-supplier --config /tmp/stake_supplier_config.yaml --from=$SUPPLIER_ADDR $TX_PARAM_FLAGS $NODE_FLAGS --keyring-backend test
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
    --config=/tmp/relayminer_config.yaml \
    # --keyring-backend=test
```

### Secure vs Non-Secure `query_node_grpc_url`

In `/tmp/relayminer_config.yaml`, you'll see that we specify an endpoint for
`query_node_grpc_url` which is TLS terminated.

If `grpc-insecure=true` then it **MUST** be an HTTP port, no TLS.

The Grove team exposed one such endpoint on one of our validators for Beta Testnet at `http://149.28.34.68:9090`.
It can be validated with `grpcurl -plaintext 149.28.34.68:9090 list`; note that the `-plaintext` flag meaning no TLS encryption.

If `grpc-insecure=false`, then it **MUST** be an HTTPS port, with TLS.

The Grove team exposed one such endpoint on one of our validators for Beta Testnet at `https://shannon-testnet-grove-grpc.beta.poktroll.com:443`.
It can be validated with `grpcurl shannon-testnet-grove-grpc.beta.poktroll.com:443 list`; note no `-plaintext` flag meaning no TLS encryption.

:::tip

You can replace both `http` and `https` with `tcp` and it should work the same.

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
poktrolld query supplier list-supplier --node https://shannon-testnet-grove-rpc.beta.poktroll.com --output json | jq
```
