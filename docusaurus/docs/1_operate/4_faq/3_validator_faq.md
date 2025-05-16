---
sidebar_position: 3
title: Validator FAQ
---

## How many validators can be staked and how many validators can produce blocks? 

An infinite number of validators can be staked, HOWEVER, only validators in the _active set_ will produce blocks. The size of the active set is dictated by the parameter `max_validators` which can be checked using `pocketd query params`. 

### How do I become an Active Validator?

Active Validators on Pocket Network are determined by the stake-weight of each validator. Only the top `N` Validator nodes by stake-weight are eligible to produce blocks. 

Each block, the chain evaluates the stake-weight of all Validators and then can promote or demote validators into the active set based on that stake-weight.

### TL;DR
👀 **TL;DR — Key Points**
- Only the top `max_validators` by total stake are active.
- New validators with higher stake can rotate out lower-staked ones.
- Validator selection is by bonded stake, not random.

🔒 **Max Validator Cap Behavior**
- If the active set is full (`max_validators` reached), new validators are created but inactive.
- They’ll be evaluated at the end of each block for possible inclusion.

⚖️ **Validator Set Rotation by Stake (Not Random)**
- Cosmos SDK sorts all bonded validators by total stake.
- Top `max_validators` are included in the active set for block signing.

🚫 **Inactive Validators**
- Validators below the threshold are not slashed or deleted.
- They remain candidates and can still receive delegations.

⏰ **When Rotation Happens**
- The validator set is recalculated at the end of each block (`staking.EndBlocker`).
- Any change in stake (new validator, delegation, slashing) can trigger reshuffling.
- A new validator with higher stake replaces a lower-staked one in the next block.

🧪 Example:
Suppose `max_validators` = 10.
```
| Rank | Validator | Stake |
|------|-----------|-------|
| 10   | J         | 100   |
| —    | New K     | 200   |
```
If `K` joins with more stake than `J`:
- `K` enters the active set.
- `J` is rotated out and becomes inactive.

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
- The [latest recommended version](../../4_develop/upgrades/4_upgrade_list.md) documentation

## What security practices should I follow?

- Never share or expose your mnemonic phrases and private keys
- Store private keys and mnemonics in secure locations
- Regularly monitor your Validator's status to prevent jailing from downtime
- When setting up your validator:
  - The `commission-rate`, `commission-max-rate`, and `commission-max-change-rate` should be expressed as decimals (e.g., `0.1` for 10%)
  - Ensure you have sufficient balance for your specified amounts in `validator.json` and delegations
