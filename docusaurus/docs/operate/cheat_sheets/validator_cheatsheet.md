---
title: Validator (~15 min)
sidebar_position: 3
---

## Validator Cheat Sheet <!-- omit in toc -->

**🖨 🍝 instructions to get you up and running with a `Validator` on Pocket Network ✅**

:::warning There is lots of scripting and some details are abstracted away

See the [Validator Walkthrough](../walkthroughs/validator_walkthrough.md) if you want to understand what's happening under the hood.

:::

## Table of Contents <!-- omit in toc -->

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

## Prerequisites

1. **CLI**: Make sure to [install the `poktrolld` CLI](../../tools/user_guide/poktrolld_cli.md).
2. **Full Node**: Make sure you have followed the [Full Node Cheat Sheet](./full_node_cheatsheet.md) to install and run a Full Node first.

:::tip `poktroll` user

If you followed [Full Node Cheat Sheet](./full_node_cheatsheet.md), you can switch
to user running the full node (which has `poktrolld` installed) like so:

```bash
su - poktroll # or a different user if you used a different name
```

:::

## Account Setup

### Create the Validator Account

Create a new key pair for the validator like so:

```bash
poktrolld keys add validator
```

### Prepare your environment

Run the following commands to set up your environment:

```bash
cat << 'EOT' > ~/.poktrollrc
export NODE="https://shannon-testnet-grove-rpc.beta.poktroll.com"
export NODE_FLAGS="--node=https://shannon-testnet-grove-rpc.beta.poktroll.com"
export TX_PARAM_FLAGS="--gas=auto --gas-prices=1upokt --gas-adjustment=1.5 --chain-id=pocket-beta --yes"
export VALIDATOR_ADDR=$(poktrolld keys show validator -a)
EOT

echo "source ~/.poktrollrc" >> ~/.bashrc
```

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

If you know someone at [Grove](https://grove.city) who maintains Beta TestNet, you
can ask them to run this command:

```bash
pkd_beta_tx tx bank send faucet_beta $VALIDATOR_ADDR 6900000000042upokt
```

:::

## Configure the Validator

### Get the Validator's PubKey

Run the following command to get the `pubkey` of your validator:

```bash
poktrolld comet show-validator
```

This will output something like:

```json
{ "@type": "/cosmos.crypto.ed25519.PubKey", "key": "YourPublicKeyHere" }
```

### Create the Validator JSON File

Create a JSON file named `validator.json` with the content below while make these changes:

- Replace the `"pubkey"` value with the output from `poktrolld comet show-validator`.
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

### Create the Validator

Run the following command to create the validator:

```bash
poktrolld tx staking create-validator ./validator.json --from=validator $TX_PARAM_FLAGS $NODE_FLAGS
```

### Verify the Validator Status

Verify the status of your validator by running:

```bash
poktrolld query staking validator $VALIDATOR_ADDR
```
