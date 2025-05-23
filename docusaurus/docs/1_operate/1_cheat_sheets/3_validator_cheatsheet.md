---
title: Validator Cheat Sheet (~15 min)
sidebar_position: 3
---

**🖨 🍝 Quick instructions to get your `Validator` running on Pocket Network ✅**

:::warning

- Lots of scripting and some details are abstracted away
- For more details, see the [Validator Walkthrough](../2_walkthroughs/3_validator_walkthrough.md)

:::

## Table of Contents <!-- omit in toc -->

- [Prerequisites](#prerequisites)
- [Account Setup](#account-setup)
  - [Create Validator Account](#create-validator-account)
  - [Set Up Environment](#set-up-environment)
  - [Fund Validator Account](#fund-validator-account)
- [Configure Validator](#configure-validator)
  - [Get Validator PubKey](#get-validator-pubkey)
  - [Create Validator JSON](#create-validator-json)
  - [Create Validator](#create-validator)
  - [Check Validator Status](#check-validator-status)

## Prerequisites

- [Install the `pocketd` CLI](../../2_explore/2_account_management/1_pocketd_cli.md)
- [Set up and run a Full Node](2_full_node_cheatsheet.md) first

:::tip `pocket` user

If you followed the Full Node Cheat Sheet, switch to the user running the full node (with `pocketd` installed):

```bash
su - pocket # or use your chosen username
```

:::

## Account Setup

### Create Validator Account

Generate a new validator key pair:

```bash
pocketd keys add validator
```

### Set Up Environment

Set environment variables:

```bash
cat << 'EOT' > ~/.pocketrc
export BETA_NODE="https://shannon-testnet-grove-rpc.beta.poktroll.com"
export BETA_NODE_FLAGS="--node=https://shannon-testnet-grove-rpc.beta.poktroll.com"
export TX_PARAM_FLAGS="--fees 200000upokt --chain-id=<CHAIN_ID>" # pocket-alpha, pocket-beta, pocket
export ADDR=$(pocketd keys show validator -a)
export VALIDATOR_ADDR=$(pocketd keys show validator -a --bech val)
EOT

echo "source ~/.pocketrc" >> ~/.bashrc
```

### Fund Validator Account

Show your validator address:

```bash
echo "Validator address: $ADDR"
```

- **Beta Testnet:** Use the [Shannon Beta TestNet faucet](https://faucet.beta.testnet.pokt.network/) to fund your account.
- **Mainnet:** Transfer funds:

```bash
pocketd tx bank send <SOURCE ADDRESS> $ADDR <AMOUNT_TO_STAKE>upokt $TX_PARAM_FLAGS
```

Check your balance:

```bash
pocketd query bank balances $ADDR
```

:::tip

Know someone at [Grove](https://grove.city) on Beta TestNet? Ask them to run:

```bash
pkd_beta_tx tx bank send faucet_beta $ADDR 6900000000042upokt [$BETA_NODE_FLAGS]
```

:::

## Configure Validator

### Get Validator PubKey

Get your validator's pubkey:

```bash
pocketd comet show-validator
```

Example output:

```json
{ "@type": "/cosmos.crypto.ed25519.PubKey", "key": "YourPublicKeyHere" }
```

### Create Validator JSON

Create `validator.json` and update:

- `"pubkey"`: Use your pubkey from above
- `"amount"`: Amount to stake (e.g., `"1000000upokt"`)
- `"moniker"`: Your validator's name (default: `validator`)
- Optionally fill in `"identity"`, `"website"`, `"security"`, `"details"`

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

### Create Validator

Register your validator:

```bash
pocketd tx staking create-validator ./validator.json --from=validator $TX_PARAM_FLAGS [$BETA_NODE_FLAGS]
```

### Check Validator Status

Check your validator status:

```bash
pocketd query staking validator $VALIDATOR_ADDR [$BETA_NODE_FLAGS]
```
