---
title: Validator Reward Playbook
sidebar_position: 2
---

This playbook provides step-by-step instructions for tracking and inspecting validator rewards on Pocket Network

:::warning TODO(@olshansk)

Remove `--grpc-insecure=false` once `pocketd` is updated

:::

## Table of Contents <!-- omit in toc -->

- [Inspecting and Retrieving Block TX Fees](#inspecting-and-retrieving-block-tx-fees)
  [Check Validator Commission](#check-outstanding-unclaimed-validator-rewards)
  [Check Outstanding Unclaimed Validator Rewards](#check-validator-commission)
  - [Withdraw Validator Commission](#withdraw-validator-commission)
  - [Check Delegator Rewards](#check-delegator-rewards)
  - [Withdraw All Rewards (Commission + Delegation)](#withdraw-all-rewards-commission--delegation)
  - [Monitor Balance Changes Over Time](#monitor-balance-changes-over-time)
- [Inspecting and Retrieving Relay Settlement Fees](#inspecting-and-retrieving-relay-settlement-fees)
  - [Check Distribution Parameters](#check-distribution-parameters)
  - [Query Community Pool](#query-community-pool)
  - [Get Comprehensive Distribution Info](#get-comprehensive-distribution-info)
  - [Withdraw All Delegation Rewards](#withdraw-all-delegation-rewards)
- [Quick Reference Playbook](#quick-reference-playbook)
  - [Daily Validator Rewards Check](#daily-validator-rewards-check)
  - [Complete Rewards Withdrawal](#complete-rewards-withdrawal)
  - [Dry Run Before Withdrawal](#dry-run-before-withdrawal)
- [Identifying the Validator Sets](#identifying-the-validator-sets)
  [Check Comet Validator Set](#view-comet-validator-set)
  - [List Bonded Validator Operator Addresses](#list-bonded-validator-operator-addresses)
  - [Convert Validator Operator Address to Account Address](#convert-validator-operator-address-to-account-address)
  - [Check Current Block Proposer](#check-current-block-proposer)

## Inspecting and Retrieving Block TX Fees

:::info Read the official Cosmos documentation for more information

Validator tx fees functionality is directly adopted from the Cosmos SDK [x/distribution](https://docs.cosmos.network/main/build/modules/distribution).

Make sure to read those docs as the primary source of truth.

:::

### Check Validator Commission

View accumulated commission that hasn't been withdrawn:

```bash
pocketd query distribution community-pool --network=main --grpc-insecure=false -o json | jq
```

# Using your validator key

```bash
pocketd query distribution commission \
 $(pocketd keys show validator --bech val -a)
```

### Check Outstanding Unclaimed Validator Rewards

View all un-withdrawn rewards for your validator:

```bash
# Using specific validator address
pocketd query distribution validator-outstanding-rewards poktvaloper1abc123...

# Using your validator key
pocketd query distribution validator-outstanding-rewards \
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

## Inspecting and Retrieving Relay Settlement Fees

### Check Distribution Parameters

Verify the current distribution parameters for relay rewards:

```bash
pocketd query tokenomics params --network=main --grpc-insecure=false -o json | jq
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

## Identifying the Validator Sets

### View Comet Validator Set

View all active validators with their voting power and proposer priority:

```bash
pocketd query comet-validator-set --network=main -o json | jq
```

_NOTE: The above are the consensus (`ed25519`) public keys of the validators,
not the account (`secp256k1`) public keys of the validator operators._

### List Bonded Validator Operator Addresses

Get all bonded (active) validators that are not jailed:

```bash
pocketd query staking validators --output json --network=main --grpc-insecure=false | \
  jq -r '.validators[] | select(.jailed != true and .status=="BOND_STATUS_BONDED") | .operator_address
```

### Convert Validator Operator Address to Account Address

Convert a validator operator address to its corresponding account address:

```bash
pocketd debug addr poktvaloper1...
```

For example:

```bash
$ pocketd debug addr poktvaloper1zppmwrdgvywrc66nn2u40ad90na9983fu9yh55
Address: [16 67 183 13 168 97 28 60 107 83 154 185 87 245 165 124 250 82 158 41]
Address (hex): 1043B70DA8611C3C6B539AB957F5A57CFA529E29
Bech32 Acc: pokt1zppmwrdgvywrc66nn2u40ad90na9983f7kh4lv
Bech32 Val: poktvaloper1zppmwrdgvywrc66nn2u40ad90na9983fu9yh55
Bech32 Con: poktvalcons1zppmwrdgvywrc66nn2u40ad90na9983fgkhtc4
```

### Check Current Block Proposer

Identify the proposer for recent blocks:

```bash
# Get latest block height
latest=$(pocketd status --network=main -o json | jq -r '.sync_info.latest_block_height')

# Check proposers for last 10 blocks
for ((h=latest; h>latest-10; h--)); do
  proposer=$(pocketd query block --type=height $h --network=main -o json | \
    jq -r '.header.proposer_address')
  echo "Block $h: $proposer"
done
```
