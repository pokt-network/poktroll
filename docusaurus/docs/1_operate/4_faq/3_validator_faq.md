---
sidebar_position: 3
title: Validator FAQ
---

## TL;DR - Quick Intro

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

**🧪 Example**:
Suppose `max_validators` = 10.

| Rank | Validator | Stake |
| ---- | --------- | ----- |
| 10   | J         | 100   |
| —    | New K     | 200   |

If `K` joins with more stake than `J`:

- `K` enters the active set.
- `J` is rotated out and becomes inactive.

## How does reward share work for Validators on Shannon?

In Shannon, we are moving away from a custom \*supplier-like rev share\* imlpementation
and defaulting to standard Cosmos best practices.

Learn more about validator stake delegation [here](https://docs.cosmos.network/main/build/modules/staking#msgdelegate).

You can read more about how delegation works [on the Cosmos Hub](https://hub.cosmos.network/main/delegators/delegator-faq), which follows similar patterns, or [this blog post](https://medium.com/@notional-ventures/staking-and-delegation-in-cosmos-db660154bcf9)

## How many validators can be staked and how many validators can produce blocks?

An infinite number of validators can be staked, HOWEVER, only validators in the _active set_ will produce blocks. The size of the active set is dictated by the parameter `max_validators` which can be checked using `pocketd query params`.

### How do I become an Active Validator?

Active Validators on Pocket Network are determined by the stake-weight of each validator. Only the top `N` Validator nodes by stake-weight are eligible to produce blocks.

Each block, the chain evaluates the stake-weight of all Validators and then can promote or demote validators into the active set based on that stake-weight.

## What is the `max_validators` parameter?

The `max_validators` parameter is a parameter that dictates the maximum number of validators that can be in the active set.

It is native to the `x/staking` module defined by the Cosmos SDK.

Note that Pocket Network has not made any changes to the `c/staking` module.

You can read more about it at [docs.cosmos.network/main/build/modules/staking](https://docs.cosmos.network/main/build/modules/staking).

## How do I check the value of the `max_validators` parameter?

```bash
pocketd query staking params
```

## How do I query the active set of validators?

```bash
pocketd query staking validators
```

## How do I delegate additional tokens to my Validator?

To increase your self-delegation or allow others to delegate to your Validator, use:

```bash
pocketd tx staking \
  delegate $VALIDATOR_ADDR <amount> \
  --from <delegator_account> \
  --fees 200000upokt \
  --chain-id=<CHAIN_ID> --node=<NODE_URL>
```

Example with specific parameters:

```bash
pocketd tx staking \
  delegate $VALIDATOR_ADDR 1000000upokt \
  --from <your_account> \
  --fees 200000upokt \
  --chain-id=<CHAIN_ID> --node=<NODE_URL>
```

## How do I unbond (undelegate) tokens from my Validator?

To unbond a portion of your staked tokens:

```bash
pocketd tx staking \
  unbond $VALIDATOR_ADDR <amount> \
  --from <delegator_account> \
  --fees 200000upokt \
  --chain-id=<CHAIN_ID> --node=<NODE_URL>
```

Example with specific parameters:

```bash
pocketd tx staking \
  unbond $VALIDATOR_ADDR 500000upokt \
  --from <your_account> \
  --fees 200000upokt \
  --chain-id=<CHAIN_ID> --node=<NODE_URL>
```

:::note Unbonding lock period

Unbonding initiates a lock period during which tokens cannot be transferred. The duration depends on network configuration which can be checked using:

```bash
pocketd query staking params -o json | jq '.params.unbonding_time'
```

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
