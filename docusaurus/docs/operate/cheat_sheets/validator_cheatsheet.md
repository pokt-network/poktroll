---
title: Validator Cheat Sheet
sidebar_position: 6
---

<<<<<<< HEAD
This cheat sheet provides quick copy-pasta instructions for staking and running a Validator node on Pocket Network.

:::info

<<<<<<< HEAD:docusaurus/docs/operate/quickstart/validator_cheatsheet.md
If you're interested in understanding everything validator related, or having full control of every
step, check out the [Validator Walkthrough](../run_a_node/validator_walkthrough.md).
=======
=======
## Validator Cheat Sheet <!-- omit in toc -->

<!-- TODO_MAINNET(@okdas, #754): Update this page with all the details. -->

>>>>>>> docs_rewrite
This cheat sheet provides quick copy-pasta like instructions for installing and
running a Validator using an automated script.

:::tip

If you're interested in understanding everything, or having full control of every
step, check out the [Validator Walkthrough](../walkthroughs/validator_walkthrough.md).
<<<<<<< HEAD
>>>>>>> 2e49d7c64 (WIP):docusaurus/docs/operate/cheat_sheets/validator_cheatsheet.md

:::

- [Prerequisites](#prerequisites)
- [Account Setup](#account-setup)
  - [Create the Validator Account](#create-the-validator-account)
  - [Prepare your environment](#prepare-your-environment)
  - [Fund the Validator account](#fund-the-validator-account)
- [Configure the Validator](#configure-the-validator)
  - [Get the Validator's PubKey](#get-the-validators-pubkey)
  - [Create the Validator JSON File](#create-the-validator-json-file)
  - [Create the Validator](#create-the-validator)
  - [Verify the Validator Status](#verify-the-validator-status)
- [Validator FAQ](#validator-faq)
  - [How do I delegate additional tokens to my validator?](#how-do-i-delegate-additional-tokens-to-my-validator)
  - [How do I unbond (undelegate) tokens from my validator?](#how-do-i-unbond-undelegate-tokens-from-my-validator)
- [Troubleshooting and Critical Notes](#troubleshooting-and-critical-notes)

## Prerequisites

1. **CLI**: Make sure to [install the `poktrolld` CLI](../user_guide/poktrolld_cli.md).
2. **Full Node**: Make sure you have followed the [Full Node Cheat Sheet](./full_node_cheatsheet.md) to install and run a Full Node first.

## Account Setup

<<<<<<< HEAD:docusaurus/docs/operate/quickstart/validator_cheatsheet.md
:::tip

if you're running a full node using the [Full Node Cheat Sheet](./full_node_cheatsheet.md), you can can switch to
the user you created in the full node setup to get access to the `poktrolld` CLI. Like this:

```bash
su - poktroll # or a different user if you used a different name
```

:::

### Create the Validator Account

Create a new key pair for the validator:

```bash
poktrolld keys add validator
```

This will generate a new address and mnemonic. **Save the mnemonic securely**.

### Prepare your environment

For convenience, we're setting several environment variables to streamline
the process of interacting with the Shannon network:

We recommend you put these in your `~/.bashrc` file:

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

### Fund the Validator account

Run the following command to get the `Validator`:

```bash
echo "Validator address: $VALIDATOR_ADDR"
```

Then use the [Shannon Beta TestNet faucet](https://faucet.beta.testnet.pokt.network/) to fund the validator account.

Afterwards, you can query the balance using the following command:

```bash
poktrolld query bank balances $VALIDATOR_ADDR $NODE_FLAGS
```

:::tip

You can find all the explorers, faucets and tools at the [tools page](../../explore/tools.md).

:::

## Configure the Validator

### Get the Validator's PubKey

To get the validator's public key, run:

```bash
poktrolld comet show-validator
```

This will output something like:

```json
{
  "@type": "/cosmos.crypto.ed25519.PubKey",
  "key": "YdlQyhjtrq9pk7afmz6oQ275L4FElzjzEJvB1fj3e1w="
}
```

**Copy the entire output; you will need it in the next step.**

### Create the Validator JSON File

Create a JSON file named `validator.json` with the following content:

```json
{
  "pubkey": {
    "@type": "/cosmos.crypto.ed25519.PubKey",
    "key": "YdlQyhjtrq9pk7afmz6oQ275L4FElzjzEJvB1fj3e1w="
  },
  "amount": "1000000upokt",
  "moniker": "YourValidatorName",
  "identity": "",
  "website": "",
  "security": "",
  "details": "",
  "commission-rate": "0.100000000000000000",
  "commission-max-rate": "0.200000000000000000",
  "commission-max-change-rate": "0.010000000000000000",
  "min-self-delegation": "1"
}
```

Make the following changes:

- Replace the `"pubkey"` value with the output from `poktrolld comet show-validator`.
- Update the `"amount"` field with the amount you wish to stake (e.g., `"1000000upokt"`).
- Set the `"moniker"` to your validator's name.
- You can optionally fill in `"identity"`, `"website"`, `"security"`, and `"details"`.

### Create the Validator

Run the following command to create the validator:

```bash
poktrolld tx staking create-validator ./validator.json --from=validator $TX_PARAM_FLAGS $NODE_FLAGS
```

This command uses the `validator.json` file to submit the `create-validator` transaction.

For example:

```bash
poktrolld tx staking create-validator ./validator.json --from=validator $TX_PARAM_FLAGS $NODE_FLAGS
```

### Verify the Validator Status

You can verify the status of your validator by running:

```bash
poktrolld query staking validator $VALIDATOR_ADDR
```

This will display information about your validator, including its status and delegation.

## Validator FAQ

### How do I delegate additional tokens to my validator?

```bash
poktrolld tx staking delegate $VALIDATOR_ADDR 1000000upokt --from your_account --chain-id=pocket-beta --gas=auto --gas-adjustment=1.5 --gas-prices=1upokt
```

### How do I unbond (undelegate) tokens from my validator?

```bash
poktrolld tx staking unbond $VALIDATOR_ADDR 500000upokt --from your_account --chain-id=pocket-beta --gas=auto --gas-adjustment=1.5 --gas-prices=1upokt
```

## Troubleshooting and Critical Notes

- Ensure your node is fully synced before attempting to create the validator.
- Keep your mnemonic and private keys secure.
- Adjust the `"amount"` in `validator.json` and delegation amounts according to your available balance.
- The `commission-rate`, `commission-max-rate`, and `commission-max-change-rate` are expressed as decimal numbers (e.g., `0.1` for 10%).
=======
1. **Run a Full Node**: Make sure you have followed the [Full Node Cheat Sheet](full_node_cheatsheet.md) to install and run a Full Node first
>>>>>>> 2e49d7c64 (WIP):docusaurus/docs/operate/cheat_sheets/validator_cheatsheet.md
=======

:::

- [Introduction](#introduction)
  - [Pre-Requisites](#pre-requisites)

## Introduction

This guide will help you install a Validator on Pocket Network,
**using helpers that abstract out some of the underlying complexity.**

### Pre-Requisites

1. **Run a Full Node**: Make sure you have followed the [Full Node Cheat Sheet](full_node_cheatsheet.md) to install and run a Full Node first
>>>>>>> docs_rewrite
