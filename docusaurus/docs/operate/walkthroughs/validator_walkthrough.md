---
title: Validator (~30 min)
sidebar_position: 4
---

## Validator Walkthrough <!-- omit in toc -->

**🧑‍🔬 detailed step-by-step instructions to get you up and running with a `Validator` on Pocket Network ✅**

:::warning This is an in-depth walkthrough

See the [Validator Cheat Sheet](../cheat_sheets/3_validator_cheatsheet.md) if you want to just copy-pasta a few commands.

:::

- [Introduction](#introduction)
- [Prerequisites \& Requirements](#prerequisites--requirements)
- [2. Account Setup](#2-account-setup)
  - [2.1. Create the Validator Account](#21-create-the-validator-account)
  - [2.2. Prepare your environment](#22-prepare-your-environment)
  - [2.3. Fund the Validator Account](#23-fund-the-validator-account)
- [3. Get the Validator's Public Key](#3-get-the-validators-public-key)
- [4. Create the Validator JSON File](#4-create-the-validator-json-file)
- [5. Create the Validator](#5-create-the-validator)
- [6. Verify the Validator Status](#6-verify-the-validator-status)

## Introduction

This guide will help you stake and run a Validator node on Pocket Network.

As a Validator, you'll be participating in the consensus of the network, validating transactions, and securing the blockchain.

## Prerequisites & Requirements

1. **CLI**: Make sure to [install the `pocketd` CLI](../../tools/user_guide/pocketd_cli.md).
2. **Synched Full Node**: Ensure you have followed the [Full Node Walkthrough](./full_node_walkthrough.md) to install and run a Full Node. Your node must be fully synced with the network before proceeding.

Ensure your node is running and fully synchronized with the network. You can check the synchronization status by running:

```bash
pocketd status
```

:::tip `pocket` user

If you followed [Full Node Walkthrough](./full_node_walkthrough.md), you can switch
to the user running the full node (which has `pocketd` installed) like so:

```bash
su - pocket # or a different user if you used a different name
```

:::

## 2. Account Setup

To become a Validator, you need a Validator account with sufficient funds to stake.

### 2.1. Create the Validator Account

Create a new key pair for your Validator account:

```bash
pocketd keys add validator
```

This will generate a new address and mnemonic.

**⚠️ Save the mnemonic securely ⚠️**.

### 2.2. Prepare your environment

For convenience, we're setting several environment variables to streamline
the process of interacting with the Shannon network:

We recommend you put these in your `~/.bashrc` file:

```bash
export BETA_NODE="https://shannon-testnet-grove-rpc.beta.poktroll.com"
export BETA_NODE_FLAGS="--node=https://shannon-testnet-grove-rpc.beta.poktroll.com"
export TX_PARAM_FLAGS="--fees 200000upokt --chain-id=<CHAIN_ID>" # pocket_beta, pocket
export ADDR=$(pocketd keys show validator -a)
export VALIDATOR_ADDR=$(pocketd keys show validator -a --bech val)
```

:::tip

Consider creating `~/.pocketrc` and appending `source ~/.pocketrc` to
your `~/.profile` (or `~/.bashrc`).

This will help keep your pocket specific environment variables separate and organized.

```bash
touch ~/.pocketrc
echo "source ~/.pocketrc" >> ~/.profile
```

:::

### 2.3. Fund the Validator Account

Run the following command to get the `Validator`:

```bash
echo "Validator address: $VALIDATOR_ADDR"
```

If you are using Beta Testnet, use the [Shannon Beta TestNet faucet](https://faucet.beta.testnet.pokt.network/) to fund the validator account.

If you are on **Mainnet** you'll need to transfer funds to the account:

```bash
pocketd tx bank send <SOURCE ADDRESS> $ADDR <AMOUNT_TO_STAKE>upokt $TX_PARAM_FLAGS
```

Afterwards, you can query the balance using the following command:

```bash
pocketd query bank balances $ADDR $NODE_FLAGS
```

:::tip

If you know someone at [Grove](https://grove.city) who maintains Beta TestNet, you
can ask them to run this command:

```bash
pkd_beta_tx tx bank send faucet_beta $ADDR 6900000000042upokt
```

:::

## 3. Get the Validator's Public Key

Your node has a unique public key associated with it, which is required for creating the Validator.

To retrieve your node's public key, run:

```bash
pocketd comet show-validator
```

This command outputs your node's public key in JSON format:

```json
{ "@type": "/cosmos.crypto.ed25519.PubKey", "key": "YourPublicKeyHere" }
```

## 4. Create the Validator JSON File

Create a JSON file named `validator.json` with the content below while make these changes:

- Replace the `"pubkey"` value with the output from `pocketd comet show-validator`.
- Update the `"amount"` field with the amount you wish to stake (e.g., `"1000000upokt"`).
- Set the `"moniker"` to your validator's name (`validator` is the default we provided).
- You can optionally fill in `"identity"`, `"website"`, `"security"`, and `"details"`.

```bash
cat << 'EOF' > validator.json
{
  "pubkey": {
    "@type": "/cosmos.crypto.ed25519.PubKey",
    "key": "YdlQyhjtrq9pk7afmz6oQ275L4FElzjzEJvB1fj3e1w="
  },
  "amount": "1000000upokt",
  "moniker": "validator",
  "identity": "",
  "website": "",
  "security": "",
  "details": "",
  "commission-rate": "0.100000000000000000",
  "commission-max-rate": "0.200000000000000000",
  "commission-max-change-rate": "0.010000000000000000",
  "min-self-delegation": "1"
}
EOF
```

## 5. Create the Validator

Run the following command to create the validator:

```bash
pocketd tx staking create-validator ./validator.json --from=validator $TX_PARAM_FLAGS
```

This command uses the `validator.json` file to submit the `create-validator` transaction.

Example with all parameters specified:

```bash
pocketd tx staking create-validator ~/validator.json --from=validator --chain-id=<CHAIN_ID> --fees 200000upokt
```

Some of the parameters you can configure include:

- `~/validator.json`: The path to your validator JSON file.
- `--from=validator`: Specifies the local key to sign the transaction.
- `--chain-id=<your-chain-id>`: Replace `<your-chain-id>` with the chain ID of the network you are joining (e.g., `pocket-beta` for testnet, `pocket` for mainnet).
- `--fees 20000upokt`: Transaction fees currently configured on both beta and mainnet.

After running the command, you should see a transaction confirmation with an output hash.

## 6. Verify the Validator Status

To verify that your Validator has been successfully created, run:

```bash
pocketd query staking validator $VALIDATOR_ADDR
```

This command displays information about your Validator, including status, tokens staked, commission rates, and more.

Ensure that the `status` field indicates that your Validator is active: `status: BOND_STATUS_BONDED`.
