---
title: Validator Rewards
sidebar_position: 2
---

# Pocket Network Validator Rewards Guide

This guide provides step-by-step instructions for validators to track and retrieve rewards on Pocket Network, including block transaction fees and relay settlement fees.

## 1. Identifying the Validator Sets

### Check Comet Validator Set

View all active validators with their voting power and proposer priority:

```bash
pocketd query comet-validator-set --network=beta -o json | jq
```

### List Bonded Validators

Get all bonded (active) validators that are not jailed:

```bash
pocketd query staking validators --output json --network=beta | \
  jq -r '.validators[] | select(.jailed != true and .status=="BOND_STATUS_BONDED") | .operator_address'
```

### Convert Validator Operator Address to Account Address

Convert a validator operator address to its corresponding account address:

```bash
pocket addr poktvaloper1...
```

### Check Current Block Proposer

Identify the proposer for recent blocks:

```bash
# Get latest block height
latest=$(pocketd status --network=beta -o json | jq -r '.sync_info.latest_block_height')

# Check proposers for last 10 blocks
for ((h=latest; h>latest-10; h--)); do
  proposer=$(pocketd query block --type=height $h --network=beta -o json | \
    jq -r '.header.proposer_address')
  echo "Block $h: $proposer"
done
```

## 2. Inspecting and Retrieving Block TX Fees

### Check Outstanding Validator Rewards

View all un-withdrawn rewards for your validator:

```bash
# Using specific validator address
pocketd query distribution validator-outstanding-rewards poktvaloper1abc123...

# Using your validator key
pocketd query distribution validator-outstanding-rewards \
  $(pocketd keys show validator --bech val -a)
```

### Check Validator Commission

View accumulated commission that hasn't been withdrawn:

```bash
# Using specific validator address
pocketd query distribution commission poktvaloper1abc123...

# Using your validator key
pocketd query distribution commission \
  $(pocketd keys show validator --bech val -a)
```

### Withdraw Validator Commission

Withdraw only your validator commission:

```bash
pocketd tx distribution withdraw-validator-commission \
  --from <validator-operator-key> \
  --chain-id pocket \
  --gas auto \
  --gas-adjustment 1.5 \
  --fees 1000000upokt
```

### Check Delegator Rewards

View rewards from delegations (including self-delegation):

```bash
# Check all delegation rewards
pocketd query distribution rewards <delegator-address>

# Check rewards from specific validator
pocketd query distribution rewards <delegator-address> <validator-operator-address>
```

### Withdraw All Rewards (Commission + Delegation)

Withdraw both commission and self-delegation rewards in one transaction:

```bash
pocketd tx distribution withdraw-rewards \
  $(pocketd keys show validator --bech val -a) \
  --commission \
  --from validator \
  --chain-id pocket \
  --gas auto \
  --gas-adjustment 1.5 \
  --fees 1000000upokt
```

### Monitor Balance Changes Over Time

Track your validator's balance changes across block heights:

```bash
# Replace with your validator account address
VALIDATOR_ADDR="pokt18rdpjl3ndma372h4503ug8cpd6kzwr8hted8wy"

# Check balance every 100 blocks
for ((h=223510; h<=225710; h+=100)); do
  echo -n "Height $h: "
  curl -s -H "x-cosmos-block-height: $h" \
    https://shannon-testnet-grove-api.beta.poktroll.com/cosmos/bank/v1beta1/balances/$VALIDATOR_ADDR | \
    jq -r '.balances[] | select(.denom=="upokt") | .amount // "0"'
done
```

## 3. Inspecting and Retrieving Relay Settlement Fees

### Check Distribution Parameters

Verify the current distribution parameters for relay rewards:

```bash
pocketd query tokenomics params --network=beta -o json | jq
```

Key parameters to check:

- `mint_allocation_percentages.proposer`: Proposer's share of new mints
- `mint_equals_burn_claim_distribution.proposer`: Proposer's share when mint equals burn

### Query Community Pool

Check the total fees accumulated in the community pool:

```bash
pocketd query distribution community-pool
```

### Get Comprehensive Distribution Info

View all distribution-related information for your validator:

```bash
pocketd query distribution validator-distribution-info poktvaloper1abc123...
```

### Withdraw All Delegation Rewards

Withdraw rewards from all your delegations at once:

```bash
pocketd tx distribution withdraw-all-rewards \
  --from <delegator-key> \
  --chain-id pocket \
  --gas auto \
  --gas-adjustment 1.5 \
  --fees 1000000upokt
```

## Quick Reference Playbook

### Daily Validator Rewards Check

```bash
# 1. Check outstanding rewards
pocketd query distribution validator-outstanding-rewards \
  $(pocketd keys show validator --bech val -a)

# 2. Check commission
pocketd query distribution commission \
  $(pocketd keys show validator --bech val -a)

# 3. Check delegation rewards
pocketd query distribution rewards \
  $(pocketd keys show validator -a)
```

### Complete Rewards Withdrawal

```bash
# Withdraw everything (commission + delegation rewards)
pocketd tx distribution withdraw-rewards \
  $(pocketd keys show validator --bech val -a) \
  --commission \
  --from validator \
  --chain-id pocket \
  --gas auto \
  --gas-adjustment 1.5 \
  --fees 1000000upokt
```

### Dry Run Before Withdrawal

Always test your transaction before executing:

```bash
pocketd tx distribution withdraw-rewards \
  $(pocketd keys show validator --bech val -a) \
  --commission \
  --from validator \
  --chain-id pocket \
  --gas auto \
  --gas-adjustment 1.5 \
  --fees 1000000upokt \
  --dry-run
```

## Important Notes

- **Outstanding rewards** show available rewards that haven't been withdrawn
- **Commission** is the validator's percentage cut from delegator rewards
- **Gas fees** are required for withdrawals - ensure sufficient balance
- **Use --dry-run** to simulate transactions before executing
- **Zero outstanding rewards** may indicate fee waiver is active or rewards already withdrawn
- Consider setting a **withdrawal address** if you want rewards sent to a different account

## Troubleshooting

If you see zero rewards:

1. Verify your validator is active and not jailed
2. Check if there's a fee waiver period active
3. Confirm you haven't already withdrawn recently
4. Ensure your validator is actually proposing blocks
