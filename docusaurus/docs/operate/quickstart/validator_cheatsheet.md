---
title: Validator Cheat Sheet
sidebar_position: 4
---

This cheat sheet provides quick copy-pasta instructions for staking and running a Validator node on Pocket Network.

- [Prerequisites](#prerequisites)
- [Account Setup](#account-setup)
  - [Create and Fund the Validator Account](#create-and-fund-the-validator-account)
- [Get the Validator's PubKey](#get-the-validators-pubkey)
- [Create the Validator JSON File](#create-the-validator-json-file)
- [Create the Validator](#create-the-validator)
- [Verify the Validator Status](#verify-the-validator-status)
- [Additional Commands](#additional-commands)
- [Notes](#notes)

## Prerequisites

1. **Run a Full Node**: Make sure you have followed the [Full Node Cheat Sheet](./full_node_cheatsheet.md) to install and run a Full Node first.

2. **Install `poktrolld` CLI**: Make sure `poktrolld` is installed and accessible from your command line.

## Account Setup

:::tip

if you're running a full node using the [Full Node Cheat Sheet](./full_node_cheatsheet.md), you can can switch to
the user you created in the full node setup to get access to the `poktrolld` CLI. Like this:

```bash
su - poktroll
```

:::

### Create and Fund the Validator Account

Create a new key pair for the validator:

```bash
poktrolld keys add validator
```

This will generate a new address and mnemonic. **Save the mnemonic securely**.

Set the validator address in an environment variable for convenience:

```bash
export VALIDATOR_ADDR=$(poktrolld keys show validator -a)
```

Fund the validator account using the TestNet faucet or by transferring tokens from another account.

Check the balance:

```bash
poktrolld query bank balances $VALIDATOR_ADDR
```

## Get the Validator's PubKey

To get the validator's public key, run:

```bash
poktrolld comet show-validator
```

This will output something like:

```json
{"@type":"/cosmos.crypto.ed25519.PubKey","key":"YdlQyhjtrq9pk7afmz6oQ275L4FElzjzEJvB1fj3e1w="}
```

Copy the entire output; you will need it in the next step.

## Create the Validator JSON File

Create a JSON file named `validator.json` with the following content:

```json
{
  "pubkey": {"@type":"/cosmos.crypto.ed25519.PubKey","key":"YdlQyhjtrq9pk7afmz6oQ275L4FElzjzEJvB1fj3e1w="},
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

- Replace the `"pubkey"` value with the output from `poktrolld comet show-validator`.
- Update the `"amount"` field with the amount you wish to stake (e.g., `"1000000upokt"`).
- Set the `"moniker"` to your validator's name.
- You can optionally fill in `"identity"`, `"website"`, `"security"`, and `"details"`.

## Create the Validator

Run the following command to create the validator:

```bash
poktrolld tx staking create-validator ~/validator.json --from validator --chain-id pocket-beta --gas=auto --gas-adjustment=1.5 --gas-prices=1upokt --yes
```

This command uses the `validator.json` file to submit the `create-validator` transaction.

**Example**:

```bash
poktrolld tx staking create-validator ~/validator.json --from validator --chain-id=pocket-beta --gas=auto --gas-prices=1upokt --gas-adjustment=1.5 --yes
```

## Verify the Validator Status

You can verify the status of your validator by running:

```bash
poktrolld query staking validator $VALIDATOR_ADDR
```

This will display information about your validator, including its status and delegation.

## Additional Commands

- **Delegate additional tokens to your validator**:

  ```bash
  poktrolld tx staking delegate $VALIDATOR_ADDR 1000000upokt --from your_account --chain-id pocket-beta --gas=auto --gas-adjustment=1.5 --gas-prices=1upokt --yes
  ```

- **Unbond (undelegate) tokens from your validator**:

  ```bash
  poktrolld tx staking unbond $VALIDATOR_ADDR 500000upokt --from your_account --chain-id pocket-beta --gas=auto --gas-adjustment=1.5 --gas-prices=1upokt --yes
  ```

- **Withdraw rewards**:

  ```bash
  poktrolld tx distribution withdraw-rewards $VALIDATOR_ADDR --commission --from validator --chain-id pocket-beta --gas=auto --gas-adjustment=1.5 --gas-prices=1upokt --yes
  ```

- **Check validator's commission and rewards**:

  ```bash
  poktrolld query distribution commission $VALIDATOR_ADDR
  poktrolld query distribution rewards $VALIDATOR_ADDR
  ```

## Notes

- Ensure your node is fully synced before attempting to create the validator.
- Keep your mnemonic and private keys secure.
- Adjust the `"amount"` in `validator.json` and delegation amounts according to your available balance.
- The `commission-rate`, `commission-max-rate`, and `commission-max-change-rate` are expressed as decimal numbers (e.g., `0.1` for 10%).

:::tip

If you're interested in understanding everything, or having full control of every
step, check out the [Validator Walkthrough](../run_a_node/validator_walkthrough.md).

:::
