---
sidebar_position: 6
title: Supplier (RelayMiner) Cheat Sheet
---

## Supplier Cheat Sheet <!-- omit in toc -->

- [Context](#context)
- [Pre-Requisites](#pre-requisites)
- [Suppliers](#suppliers)
  - [How do I query for all existing onchain Suppliers?](#how-do-i-query-for-all-existing-onchain-suppliers)
  - [How do I stake an onchain Supplier?](#how-do-i-stake-an-onchain-supplier)
  - [Supplier Configuration](#supplier-configuration)
  - [Supplier Transaction](#supplier-transaction)
- [How do I learn more about interacting with Suppliers?](#how-do-i-learn-more-about-interacting-with-suppliers)
  - [Supplier Transactions](#supplier-transactions)
  - [Supplier Queries](#supplier-queries)
- [RelayMiners](#relayminers)
  - [Retrieve the source code](#retrieve-the-source-code)

### Context

This document is a cheat sheet to get you quickly started with two things:

1. Staking an onchain `Supplier`
2. Deploying an offchain `RelayMiner`

By the end of it, you should be able to serve Relays onchain.

:::tip

It is intended to be a < 10 minute quick copy-pasta.

If you're interested in spending hours reading and understanding how things work,
please see the [Supplier Walkthrough](./../run_a_node/supplier_walkthrough.md)

:::

### Pre-Requisites

1. Make sure to [install the `poktrolld` CLI](../user_guide/install.md).
2. Make sure you know how to [create and fund a new `account`](../user_guide/create-new-wallet.md).
3. You have either [staked a new `service` or found an existing](./service_cheatsheet.md).
4. `[Optional]` You can run things locally or have dedicated long-running hardware. See the [Docker Compose Cheat Sheet](./docker_compose_debian_cheatsheet#deploy-your-server) if you're interested in the latter.

### Suppliers

#### How do I query for all existing onchain Suppliers?

Then, you can query for all services like so:

```bash
poktrolld query supplier list-supplier --node https://shannon-testnet-grove-rpc.beta.poktroll.com --output json | jq
```

#### How do I stake an onchain Supplier?

:::tip

For an in-depth look at how to stake a supplier, see the [Supplier configuration docs](./../configs/supplier_staking_config.md).

:::

The following is a very quick and simple way to get you started by staking for
Pocket Network's Morse service on Shannon using a public RPC endpoint provided by
[Liquify](https://liquify.com/).

```yaml
owner_address: pokt1v6ap5mmaaldw35vrhtmwm6uxr9dn3jz8zj9cmk
operator_address: pokt1v6ap5mmaaldw35vrhtmwm6uxr9dn3jz8zj9cmk
stake_amount: 1000069upokt
default_rev_share_percent:
  pokt1v6ap5mmaaldw35vrhtmwm6uxr9dn3jz8zj9cmk: 100
services:
  - service_id: "morse"
    endpoints:
      - publicly_exposed_url: https://pocket-rpc.liquify.com
        rpc_type: JSON_RPC
```

```
poktrolld tx supplier stake-supplier --config /var/folders/th/667_sx1j13343j4_k93ppf380000gn/T/tmp0gqsnj4k --from user_key_2ui31x --yes --output json --node tcp://127.0.0.1:26657 --chain-id poktroll --home /Users/olshansky/workspace/pocket/poktroll/localnet/poktrolld --keyring-backend test
```

Here is an example of the output on Beta TestNet as of writing this document:

```json
default_signing_key_names: [user_key_ste8bz]
smt_store_path: /tmp/poktroll/smt
metrics:
  enabled: true
  addr: :9091
pocket_node:
  query_node_rpc_url: https://shannon-testnet-grove-rpc.beta.poktroll.com
  query_node_grpc_url: https://shannon-testnet-grove-grpc.beta.poktroll.com
  tx_node_rpc_url: https://shannon-testnet-grove-rpc.beta.poktroll.com
suppliers:
  - service_id: svc_8ymf38
    listen_url: http://localhost:8500
    service_config:
      backend_url: http://localhost:8547
      publicly_exposed_endpoints:
        - localhost
pprof:
  enabled: false
  addr: localhost:6060
ping:
  enabled: false
  addr: localhost:8082
```

#### Supplier Configuration

#### Supplier Transaction

poktrolld tx supplier stake-supplier --config /var/folders/th/667_sx1j13343j4_k93ppf380000gn/T/tmp0gqsnj4k --from user_key_2ui31x --yes --output json --node tcp://127.0.0.1:26657 --chain-id poktroll --home /Users/olshansky/workspace/pocket/poktroll/localnet/poktrolld --keyring-backend test
You can use the `ad d-service` command to create a new service like so:

```bash
poktrolld tx service add-service ${SERVICE_ID} "${SERVICE_NAME_OR_DESCRIPTION}" ${COMPUTE_UNITS_PER_RELAY} --from ${SERVICE_OWNER}
```

Here is a concrete copy-pasta assuming you have created and funded a new account called `$USER`:

```bash
poktrolld tx service add-service "svc-$USER" "service description for $USER" 69 \
    --node https://shannon-testnet-grove-rpc.beta.poktroll.com \
    --fees 1upokt --from $USER --chain-id pocket-beta
```

Optionally, you can add some more flags to be ultra-verbose about your local environment:

```bash
poktrolld tx service add-service "svc-$USER" "service description for $USER" 69 \
    --node https://shannon-testnet-grove-rpc.beta.poktroll.com \
    --fees 1upokt --from $USER --chain-id pocket-beta \
    --home ~/.poktroll --keyring-backend test \
    --yes --output json
```

### How do I learn more about interacting with Suppliers?

#### Supplier Transactions

```bash
poktrolld tx supplier -h
```

#### Supplier Queries

```bash
poktrolld query supplier -h
```

### RelayMiners

#### Retrieve the source code

```bash
mkdir ~/workspace && cd ~/workspace
git clone https://github.com/pokt-network/poktroll-docker-compose-example.git
cd poktroll-docker-compose-example
```
