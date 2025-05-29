---
sidebar_position: 4
title: Supplier & RelayMiner Cheat Sheet (~25 minutes)
---

import ReactPlayer from "react-player";

:::warning 🖨 🍝 with Scripted Abstractions 🍝 🖨

Stake an onchain `Supplier` and run an offchain `RelayMiner` in less than an hour, without deep explanations.

:::

---

## Table of Contents <!-- omit in toc -->

- [High Level Architecture Diagram](#high-level-architecture-diagram)
- [20 Minute Video Walkthrough](#20-minute-video-walkthrough)
- [Prerequisites](#prerequisites)
  - [What will you do in this cheatsheet?](#what-will-you-do-in-this-cheatsheet)
- [Account Setup](#account-setup)
  - [1. Create Supplier account](#1-create-supplier-account)
  - [2. Prepare your environment](#2-prepare-your-environment)
  - [3. Fund the Supplier account](#3-fund-the-supplier-account)
- [Supplier Configuration](#supplier-configuration)
  - [1. Get your public URL](#1-get-your-public-url)
  - [2. Configure your Supplier](#2-configure-your-supplier)
  - [3. Stake your Supplier](#3-stake-your-supplier)
- [RelayMiner Configuration](#relayminer-configuration)
  - [(Optional) Start the anvil node](#optional-start-the-anvil-node)
  - [1. Configure the RelayMiner](#1-configure-the-relayminer)
  - [2. Start the RelayMiner](#2-start-the-relayminer)
  - [3. Test the RelayMiner](#3-test-the-relayminer)

## High Level Architecture Diagram

```mermaid
flowchart TB
    %% Set default styling for all nodes
    classDef default fill:#f9f9f9,stroke:#333,stroke-width:1px,color:black;

    %% Position User at top
    User([User]):::userClass

    %% Gateway layer - GREEN background
    subgraph Gateway["🌿 Grove Gateway (Offchain)"]
        direction TB
        SSDK1["Shannon SDK"]:::gatewayClass
        PSDK["PATH SDK"]:::gatewayClass
        PATHC{{"PATH Config File"}}:::configClass
        GATEWAYKEY[["Gateway Private Key"]]:::keyClass
        APPKEY1[["App Private Key(s)"]]:::keyClass
    end

    %% pocketd CLI layer - ORANGE background
    %% subgraph pocketd["pocketd CLI (Offchain)"]
    %%     direction TB
    %%     SSDK2["Shannon SDK"]:::pocketdClass
    %%     KEYSTORE[("Keystore Database")]:::dbClass
    %% end

    %% Blockchain layer - BLUE background
    subgraph Blockchain["🌀 Pocket Network (Onchain)"]
        direction LR
        subgraph Records["Network State"]
            APPDB[("Application Registry")]:::dbClass
            GWDB[("Gateway Registry")]:::dbClass
            SUPDB[("Supplier Registry")]:::dbClass

            APPCONFIG1{{"App Config 1"}}:::configClass
            APPCONFIGN{{"App Config N"}}:::configClass
            GWCONFIG{{"Gateway Config"}}:::configClass
        end
    end

    %% Supplier layer
    subgraph Suppliers["Suppliers"]
        direction TB
        S1["Supplier Node 1"]:::supplierClass
        SN["Supplier Node N"]:::supplierClass
        SUPKEY1[["Supplier Private Key"]]:::keyClass
        SUPKEYN[["Supplier Private Key"]]:::keyClass
    end

    %% Connections
    %% User <-->|"RPC Request/Response"| pocketd
    User <-->|"RPC Request/Response"| Gateway

    Gateway -.->|"References"| GWCONFIG
    Gateway -.->|"References"| APPCONFIGN
    %% pocketd -.->|"References"| APPCONFIG1

    %% pocketd <-->|"Signed Relay Request/Response"| Suppliers
    Suppliers <-->|"Signed Relay Request/Response"| Gateway

    %% Connect configs to databases
    APPCONFIG1 --- APPDB
    APPCONFIGN --- APPDB
    GWCONFIG --- GWDB

    %% Connect keys to suppliers
    S1 --- SUPKEY1
    SN --- SUPKEYN

    %% Define custom classes with specified colors
    classDef userClass fill:#f0f0f0,stroke:#333,stroke-width:2px,color:black;
    classDef gatewayClass fill:#e8f5e8,stroke:#4caf50,stroke-width:2px,color:black;
    classDef pocketdClass fill:#fff3e0,stroke:#ff8f00,stroke-width:2px,color:black;
    classDef blockchainClass fill:#e3f2fd,stroke:#2196f3,stroke-width:2px,color:black;
    classDef supplierClass fill:#fff3e0,stroke:#ff9800,stroke-width:2px,color:black;
    classDef keyClass fill:#ffebee,stroke:#d32f2f,stroke-width:1px,color:black;
    classDef configClass fill:#f3e5f5,stroke:#7b1fa2,stroke-width:1px,color:black;
    classDef dbClass fill:#e0f2f1,stroke:#00695c,stroke-width:1px,color:black;

    %% Apply classes to subgraphs
    class Blockchain blockchainClass
    class Gateway gatewayClass
    class Suppliers supplierClass

```

## 20 Minute Video Walkthrough

<ReactPlayer
  playing={false}
  controls
  url="https://github.com/user-attachments/assets/bafd0b3e-4968-4e92-ba8a-41b618633455"
/>

## Prerequisites

- [Install `pocketd` CLI](../../2_explore/2_account_management/1_pocketd_cli.md)
- [Create and fund account](../../2_explore/2_account_management/2_create_new_account_cli.md)
- [Stake or find a `service`](1_service_cheatsheet.md)
- [Review hardware requirements](../4_faq/6_hardware_requirements.md)

:::note Optional Vultr Setup

The instructions on this page assume you have experience maintaining backend services.

You can reference the [Vultr Playbook](../5_playbooks/1_vultr.md) for a quick guide on how to set up a server with Vultr.

:::

### What will you do in this cheatsheet?

1. Stake a `Supplier` (i.e. onchain record)
2. Deploy a `RelayMiner` (i.e. offchain coprocessor)
3. Serve relays
4. Claim rewards
5. Submit proofs
6. Earn rewards for onchain services

## Account Setup

### 1. Create Supplier account

```bash
pocketd keys add supplier
```

### 2. Prepare your environment

Create the following environment variables:

```bash
cat > ~/.pocketrc << EOF
export SUPPLIER_ADDR=$(pocketd keys show supplier -a)
export TX_PARAM_FLAGS="--gas=auto --gas-prices=1upokt --gas-adjustment=1.5 --yes"
export BETA_NODE_FLAGS="--chain-id=pocket-beta --node=https://shannon-testnet-grove-rpc.beta.poktroll.com"
export BETA_NETWORK="pocket-beta"
export BETA_RPC_URL="https://shannon-testnet-grove-rpc.beta.poktroll.com"
export BETA_GRPC_URL="https://shannon-testnet-grove-grpc.beta.poktroll.com:443"
export BETA_GRPC_URL_RAW="shannon-testnet-grove-grpc.beta.poktroll.com:443"
EOF
```

And source them in your shell:

```bash
echo "source ~/.pocketrc" >> ~/.profile
source ~/.profile
```

### 3. Fund the Supplier account

1. Retrieve your Supplier address:

   ```bash
   echo "Supplier address: $SUPPLIER_ADDR"
   ```

2. Fund your account by going to [Shannon Beta TestNet faucet](https://faucet.beta.testnet.pokt.network/).

3. Check balance:

   ```bash
   pocketd query bank balances $SUPPLIER_ADDR $BETA_NODE_FLAGS
   ```

:::tip 🌿 Grove employees only

<details>

<summary>`pkd` helpers</summary>

```bash
# Fund your account
pkd_beta_fund $SUPPLIER_ADDR

# Check balance
pkd_beta_query bank balances $SUPPLIER_ADDR
```

</details>

:::

## Supplier Configuration

For more details on supplier configurations, see the full [supplier config docs](../3_configs/3_supplier_staking_config.md).

### 1. Get your public URL

Retrieve your external IP:

```bash
EXTERNAL_IP=$(curl -4 ifconfig.me/ip)
```

Pick a public port to open (e.g. 8545):

```bash
sudo ufw allow 8545/tcp
```

Your supplier will be accessible at:

```bash
echo http://${EXTERNAL_IP}:8545
```

### 2. Configure your Supplier

Prepare the stake supplier config:

```bash
cat <<🚀 > /tmp/stake_supplier_config.yaml
owner_address: $SUPPLIER_ADDR
operator_address: $SUPPLIER_ADDR
stake_amount: 1000069upokt
default_rev_share_percent:
  $SUPPLIER_ADDR: 100
services:
  - service_id: "anvil" # change if not using Anvil
    endpoints:
      - publicly_exposed_url: http://$EXTERNAL_IP:8545 # must be public
        rpc_type: JSON_RPC
🚀
```

:::warning Replace `service_id`

The example uses `service_id: anvil`.
Use your own service_id or [create a new one](1_service_cheatsheet.md).

:::

### 3. Stake your Supplier

Submit the staking transaction:

```bash
pocketd tx supplier stake-supplier \
  --config /tmp/stake_supplier_config.yaml \
  --from=$SUPPLIER_ADDR $TX_PARAM_FLAGS $BETA_NODE_FLAGS
```

And check the status onchain:

```bash
pocketd query supplier show-supplier $SUPPLIER_ADDR $BETA_NODE_FLAGS
```

## RelayMiner Configuration

See [RelayMiner config docs](../3_configs/4_relayminer_config.md) for all options.

### (Optional) Start the anvil node

If using `service_id: anvil`, run a local Anvil node:

<details>
<summary>How to run Anvil</summary>

```bash
curl -L https://foundry.paradigm.xyz | bash
source ~/.foundry/bin
foundryup
anvil --port 8546
```

Test:

```bash
curl -X POST http://127.0.0.1:8546 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": 1, "method": "eth_blockNumber", "params": []}'
```

</details>

### 1. Configure the RelayMiner

Prepare the RelayMiner (i.e. the offchain co-processor) config:

```bash
cat <<🚀 > /tmp/relayminer_config.yaml
default_signing_key_names:
  - supplier
smt_store_path: $HOME/.pocket/smt
pocket_node:
  query_node_rpc_url: $BETA_RPC_URL
  query_node_grpc_url: $BETA_GRPC_URL
  tx_node_rpc_url: $BETA_RPC_URL
suppliers:
  - service_id: "anvil" # change if not using Anvil
    service_config:
      backend_url: "http://127.0.0.1:8546" # change if not using Anvil
    listen_url: http://0.0.0.0:8545 # must match Supplier's public URL
metrics:
  enabled: false
  addr: :9090
pprof:
  enabled: false
  addr: :6060
🚀
```

### 2. Start the RelayMiner

Start the RelayMiner (i.e. the offchain co-processor) server:

```bash
pocketd \
  relayminer start \
  --grpc-insecure=false \
  --log_level=debug \
  --config=/tmp/relayminer_config.yaml \
  --chain-id=$BETA_NETWORK
```

### 3. Test the RelayMiner

After following the instructions in the [Gateway cheatsheet](5_gateway_cheatsheet.md), you can use your `Application` to send a relay request to your supplier assuming it is staked for the same service:

The following is an example of a relay request to an Anvil (i.e. EVM) node:

```bash
pocketd relayminer relay \
  --app=$APP_ADDR \
  --supplier=$SUPPLIER_ADDR \
  --node=$BETA_RPC_URL \
  --grpc-addr=$BETA_GRPC_URL_RAW \
  --grpc-insecure=false \
  --payload="{\"jsonrpc\": \"2.0\", \"id\": 1, \"method\": \"eth_blockNumber\", \"params\": []}"
```

<details>
<summary>*tl;dr staking an application for `anvil`*</summary>

**Create an application:**

```bash
pocketd keys add application
```

Fund it (faucet or other).

**Prepare the stake config:**

```bash
cat <<🚀 > /tmp/stake_app_config.yaml
stake_amount: 100000000upokt
service_ids:
  - "anvil"
🚀
```

**Stake it:**

```bash
pocketd tx application stake-application \
  --config=/tmp/stake_app_config.yaml \
  --from=$(pocketd keys show application -a) $TX_PARAM_FLAGS $BETA_NODE_FLAGS
```

**Check the staking status:**

```bash
pocketd query application show-application $(pocketd keys show application -a) $BETA_NODE_FLAGS
```

</details>
