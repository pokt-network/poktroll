---
title: Validator Walkthrough
sidebar_position: 4
---

<<<<<<< HEAD
This walkthrough provides detailed step-by-step instructions to stake and run a Validator node on Pocket Network.

:::tip

<<<<<<< HEAD:docusaurus/docs/operate/run_a_node/validator_walkthrough.md
If you're interested in a simple guide with _copy-pasta_ of a few commands to get started, check out the [Validator Cheat Sheet](../quickstart/validator_cheatsheet.md) instead.
=======
If you're comfortable using an automated scripts, or simply want to _copy-pasta_ a
few commands to get started, check out the [Validator Cheat Sheet](../cheat_sheet/validator_cheatsheet.md).
>>>>>>> 2e49d7c64 (WIP):docusaurus/docs/operate/walkthroughs/validator_walkthrough.md
=======
## Validator Walkthrough <!-- omit in toc -->

<!-- TODO_MAINNET(@okdas, #754): Update this page with all the details. -->

This walkthrough provides a detailed step-by-step instructions to run a validator node for Pocket Network.

:::tip

If you're comfortable using an automated scripts, or simply want to _copy-pasta_ a
few commands to get started, check out the [Validator Cheat Sheet](../cheat_sheet/validator_cheatsheet.md).
>>>>>>> docs_rewrite

:::

- [Introduction](#introduction)
<<<<<<< HEAD
- [Prerequisites](#prerequisites)
- [1. Run a Full Node](#1-run-a-full-node)
- [2. Account Setup](#2-account-setup)
  - [2.1. Create the Validator Account](#21-create-the-validator-account)
  - [2.2. Prepare your environment](#22-prepare-your-environment)
  - [2.3. Fund the Validator Account](#23-fund-the-validator-account)
- [3. Get the Validator's Public Key](#3-get-the-validators-public-key)
- [4. Create the Validator JSON File](#4-create-the-validator-json-file)
- [5. Create the Validator](#5-create-the-validator)
- [6. Verify the Validator Status](#6-verify-the-validator-status)
- [7. Additional Commands](#7-additional-commands)
- [Notes](#notes)

## Introduction

This guide will help you stake and run a Validator node on Pocket Network, from scratch, manually, **giving you control over each step of the process**.

As a Validator, you'll be participating in the consensus of the network, validating transactions, and securing the blockchain.

<<<<<<< HEAD:docusaurus/docs/operate/run_a_node/validator_walkthrough.md
## Prerequisites

**Run a Full Node**: Ensure you have followed the [Full Node Walkthrough](./full_node_walkthrough.md) to install and run a Full Node. Your node must be fully synced with the network before proceeding.

## 1. Run a Full Node

Before becoming a Validator, you need to run a Full Node. If you haven't set up a Full Node yet, please follow the [Full Node Walkthrough](./full_node_walkthrough.md) to install and configure your node.

:::tip

if you're already running a full node using the [Full Node Walkthrough](./full_node_walkthrough.md), you can can switch to
the user you created in the full node setup to get access to the `poktrolld` CLI. Like this:

```bash
su - poktroll # or a different user if you used a different name
```

:::

Ensure your node is running and fully synchronized with the network. You can check the synchronization status by running:

```bash
poktrolld status
```

## 2. Account Setup

To become a Validator, you need a Validator account with sufficient funds to stake.

### 2.1. Create the Validator Account

Create a new key pair for your Validator account:

```bash
poktrolld keys add validator
```

### 2.2. Prepare your environment

For convenience, we're setting several environment variables to streamline
the process of interacting with the network:

```bash
export NODE="https://shannon-testnet-grove-rpc.beta.poktroll.com"
export NODE_FLAGS="--node=https://shannon-testnet-grove-rpc.beta.poktroll.com"
export TX_PARAM_FLAGS="--gas=auto --gas-prices=1upokt --gas-adjustment=1.5 --chain-id=pocket-beta --yes"
export VALIDATOR_ADDR=$(poktrolld keys show validator -a)
```

:::tip
As an alternative to appending directly to `~/.bashrc`, you can put the above
in a special `~/.poktrollrc` and add `source ~/.poktrollrc` to
your `~/.profile` (or `~/.bashrc`) file for a cleaner organization.
:::

### 2.3. Fund the Validator Account

Run the following command to get the `Validator`:

```bash
echo "Validator address: $VALIDATOR_ADDR"
```

Then use the [Shannon Beta TestNet faucet](https://faucet.beta.testnet.pokt.network/) to fund the validator account.

Check the balance of your Validator account:

```bash
poktrolld query bank balances $VALIDATOR_ADDR $NODE_FLAGS
```

:::tip
You can find all the explorers, faucets and tools at the [tools page](../../explore/tools.md).
:::

## 3. Get the Validator's Public Key

Your node has a unique public key associated with it, which is required for creating the Validator.

To retrieve your node's public key, run:

```bash
poktrolld comet show-validator
```

This command outputs your node's public key in JSON format:

```json
{"@type":"/cosmos.crypto.ed25519.PubKey","key":"YourPublicKeyHere"}
```

- Copy the entire output (including `"@type"` and `"key"`), as you'll need it for the next step.

## 4. Create the Validator JSON File

Create a JSON file named `validator.json` in your home directory (or any convenient location), which contains the information required to create your Validator.

```bash
nano ~/validator.json
```

Paste the following content into `validator.json`, replacing placeholders with your information:

```json
{
  "pubkey": {"@type":"/cosmos.crypto.ed25519.PubKey","key":"YourPublicKeyHere"},
  "amount": "1000000upokt",
  "moniker": "YourValidatorName",
  "identity": "",
  "website": "",
  "security": "",
  "details": "",
  "commission-rate": "0.10",
  "commission-max-rate": "0.20",
  "commission-max-change-rate": "0.01",
  "min-self-delegation": "1"
}
```

- **Replace** `"YourPublicKeyHere"` with the `"key"` value from `poktrolld comet show-validator`.
- **Update** `"amount"` with the amount you wish to stake (e.g., `"1000000upokt"`). Ensure this amount is less than or equal to your account balance.
- **Set** `"moniker"` to your desired Validator name. This is how your Validator will appear to others.
- **Optional**: Fill in `"identity"`, `"website"`, `"security"`, and `"details"` if you wish to provide additional information about your Validator.

Save and close the file.

## 5. Create the Validator

Now, you are ready to create your Validator on the network.

Run the following command:

```bash
poktrolld tx staking create-validator ~/validator.json --from=validator $TX_PARAM_FLAGS $NODE_FLAGS
```

- **Parameters**:
  - `~/validator.json`: The path to your validator JSON file.
  - `--from=validator`: Specifies the local key to sign the transaction.
  - `--chain-id=<your-chain-id>`: Replace `<your-chain-id>` with the chain ID of the network you are joining (e.g., `pocket-beta` for testnet).
  - `--gas=auto`: Automatically estimate gas required for the transaction.
  - `--gas-adjustment=1.5`: Adjust the estimated gas by a factor (can help prevent out-of-gas errors).
  - `--gas-prices=1upokt`: Set the gas price; adjust as needed based on network conditions.

**Example**:

```bash
poktrolld tx staking create-validator ~/validator.json --from=validator --chain-id=pocket-beta --gas=auto --gas-adjustment=1.5 --gas-prices=1upokt
```

After running the command, you should see a transaction confirmation with an output hash.

## 6. Verify the Validator Status

To verify that your Validator has been successfully created, run:

```bash
poktrolld query staking validator $VALIDATOR_ADDR
```

This command displays information about your Validator, including status, tokens staked, commission rates, and more.

Ensure that the `status` field indicates that your Validator is active.

## 7. Additional Commands

Here are some useful commands for managing your Validator:

- **Delegate additional tokens to your Validator**:

  If you wish to increase your self-delegation or others want to delegate to your Validator, use:

  ```bash
  poktrolld tx staking delegate $VALIDATOR_ADDR <amount> --from <delegator_account> $TX_PARAM_FLAGS $NODE_FLAGS
  ```

  Replace `<amount>` with the amount to delegate (e.g., `1000000upokt`) and `<delegator_account>` with the name of the key in your keyring.

- **Unbond (undelegate) tokens from your Validator**:

  To unbond a portion of your staked tokens:

  ```bash
  poktrolld tx staking unbond $VALIDATOR_ADDR <amount> --from <delegator_account> $TX_PARAM_FLAGS $NODE_FLAGS
  ```

  Note that unbonding tokens initiates an unbonding period during which the tokens are locked. The unbonding period duration depends on the network configuration.

## Notes

- **Node Synchronization**: Your Full Node must be fully synchronized with the network before creating the Validator. Use `poktrolld status` to check synchronization status.

- **Security**: Keep your mnemonic phrases and private keys secure. Do not share them or store them in insecure locations.

- **Monitoring**: Regularly monitor your Validator's status to ensure it remains active and does not get jailed due to downtime or misbehavior.

- **Upgrades**: Keep your node software up-to-date. Follow upgrade notifications in Pocket Network's [Discord](https://discord.com/invite/pocket-network) and ensure your node is running the [latest recommended version](../../protocol/upgrades/upgrade_list.md).

---

Congratulations! You have successfully set up and run a Validator on Pocket Network. Remember to stay engaged with the community and keep your node running smoothly to contribute to the network's security and decentralization.
=======
1. **Run a Full Node**: Make sure you have followed the [Full Node Walkthrough](full_node_walkthrough.md) to install and run a Full Node first
>>>>>>> 2e49d7c64 (WIP):docusaurus/docs/operate/walkthroughs/validator_walkthrough.md
=======
- [Pre-Requisites](#pre-requisites)

## Introduction

This guide will help you install a Validator on Pocket Network, from scratch, manually,
**giving you control over each step of the process**.

## Pre-Requisites

1. **Run a Full Node**: Make sure you have followed the [Full Node Walkthrough](full_node_walkthrough.md) to install and run a Full Node first
>>>>>>> docs_rewrite
