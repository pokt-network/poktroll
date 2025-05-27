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

    %% Define custom classes
    classDef userClass fill:#e1f5fe,stroke:#01579b,stroke-width:2px,color:black;
    classDef blockchainClass fill:#e1f5fe,stroke:#01579b,stroke-width:2px,color:black;
    classDef relayMinerClass fill:#fff8e1,stroke:#ff8f00,stroke-width:2px,color:black;
    classDef databaseClass fill:#f3e5f5,stroke:#6a1b9a,stroke-width:2px,color:black;
    classDef supplierRecordsClass fill:#bbdefb,stroke:#1976d2,stroke-width:2px,color:black;

    %% Position User at top
    User([User]):::userClass

    %% Position RelayMiner on left and Blockchain on right
    subgraph Operator["Operator (Offchain)"]
        direction TB
        CP["RelayMiner<br>Co-processor"]
        subgraph BS["Backend Service"]
            SRV["Server"]
            SRC["Open Service"]:::databaseClass
            DB2[(Open Database)]:::databaseClass
        end
    end


    %% User flow
    User -->|"Signed Relay<br>**Request**"| CP
    CP -->|"Data/Service<br>**Request**"| SRV
    SRV -.- DB2
    SRV -.- SRC
    SRV -->|"Data/Service<br>**Response**"| CP
    CP -->|"Signed Relay<br>**Response**"| User

    %% Connection between RelayMiner and Blockchain
    CP -..- SC
    DB --- SC
    DB --- SCN


    subgraph Blockchain["Pocket Network (Onchain)"]
        direction TB
        DB[(All Supplier Records)]:::supplierRecordsClass
        SCN[["Supplier Config N"]]
        SC[["Supplier Config 1"]]
    end

    %% Apply classes to subgraphs
    class Blockchain blockchainClass;
    class Operator relayMinerClass;
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
export BETA_RPC_URL="https://shannon-testnet-grove-rpc.beta.poktroll.com"
export BETA_GRPC_URL="https://shannon-testnet-grove-grpc.beta.poktroll.com:443"
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
  --config=/tmp/relayminer_config.yaml
```

### 3. Test the RelayMiner

After following the instructions in the [Gateway cheatsheet](5_gateway_cheatsheet.md), you can use your `Application` to send a relay request to your supplier assuming it is staked for the same service:

```bash
pocketd relayminer relay \
  --app=pokt12fj3xlqg6d20fl4ynuejfqd3fkqmq25rs3yf7g \
  --supplier=pokt1hwed7rlkh52v6u952lx2j6y8k9cn5ahravmzfa \
  --node=$BETA_RPC_URL \
  --grpc-addr=$BETA_GRPC_URL \
  --grpc-insecure=false \
  --payload="{\"jsonrpc\": \"2.0\", \"id\": 1, \"method\": \"eth_blockNumber\", \"params\": []}"
```

<details>
<summary>*tl;dr staking an application for `anvil`*</summary>

```bash
# Create an application
pocketd keys add application


# Fund it (faucet or other)

# Prepare the stake config
cat <<🚀 > /tmp/stake_app_config.yaml
stake_amount: 100000000upokt
service_ids:
  - "anvil"
🚀

# Stake it
pocketd tx application stake-application \
  --config=/tmp/stake_app_config.yaml \
  --from=$(pocketd keys show application -a) $TX_PARAM_FLAGS $BETA_NODE_FLAGS

# Check status
pocketd query application show-application $(pocketd keys show application -a) $BETA_NODE_FLAGS
```

</details>
