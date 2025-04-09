---
sidebar_position: 2
title: Validator FAQ
---

## How do I delegate additional tokens to my Validator?

To increase your self-delegation or allow others to delegate to your Validator, use:

```bash
pocketd tx staking delegate $VALIDATOR_ADDR <amount> --from <delegator_account> $TX_PARAM_FLAGS $NODE_FLAGS
```

Example with specific parameters:

```bash
pocketd tx staking delegate $VALIDATOR_ADDR 1000000upokt --from your_account --chain-id=pocket-beta --gas=auto --gas-adjustment=1.5 --gas-prices=1upokt
```

## How do I unbond (undelegate) tokens from my Validator?

To unbond a portion of your staked tokens:

```bash
pocketd tx staking unbond $VALIDATOR_ADDR <amount> --from <delegator_account> $TX_PARAM_FLAGS $NODE_FLAGS
```

Example with specific parameters:

```bash
pocketd tx staking unbond $VALIDATOR_ADDR 500000upokt --from your_account --chain-id=pocket-beta --gas=auto --gas-adjustment=1.5 --gas-prices=1upokt
```

:::note Unbonding lock period

Unbonding initiates a lock period during which tokens cannot be transferred. The duration depends on network configuration.

:::

## How do I check if my node is synchronized?

Use the following command to check your node's synchronization status:

```bash
pocketd status
```

:::note Synchronization status

Ensure that the `sync_info.catching_up` field is `false` to confirm that your node is fully synchronized.

Your Full Node must be fully synchronized before creating a Validator.

:::

## How do I stay updated with network upgrades?

**Monitor and follow**:

- Upgrade notifications in [Pocket Network's Discord](https://discord.com/invite/pocket-network)
- The [latest recommended version](../upgrades/1_upgrade_list.md) documentation

## What security practices should I follow?

- Never share or expose your mnemonic phrases and private keys
- Store private keys and mnemonics in secure locations
- Regularly monitor your Validator's status to prevent jailing from downtime
- When setting up your validator:
  - The `commission-rate`, `commission-max-rate`, and `commission-max-change-rate` should be expressed as decimals (e.g., `0.1` for 10%)
  - Ensure you have sufficient balance for your specified amounts in `validator.json` and delegations
