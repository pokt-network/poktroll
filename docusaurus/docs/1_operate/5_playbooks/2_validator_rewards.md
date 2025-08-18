---
title: Validator Reward Playbook
sidebar_position: 2
---

This playbook provides step-by-step instructions for tracking and inspecting validator rewards on Pocket Network

:::warning TODO(@olshansk)

Remove `--grpc-insecure=false` once `pocketd` is updated

:::

## Table of Contents <!-- omit in toc -->

- [Identifying the Validator Sets](#identifying-the-validator-sets)
  - [1. View CometBFT Consensus Validator Set (`ed25519`)](#1-view-cometbft-consensus-validator-set-ed25519)
  - [2. List CosmosSDK Bonded Validator Operator Addresses (`secp256k1`)](#2-list-cosmossdk-bonded-validator-operator-addresses-secp256k1)
  - [3. Check Current Block Proposer (encoded `ed25519`)](#3-check-current-block-proposer-encoded-ed25519)
- [Inspecting and Retrieving Block TX Fees](#inspecting-and-retrieving-block-tx-fees)
  - [Check Validator Commission](#check-validator-commission)
  - [Using your validator key](#using-your-validator-key)
  - [Check Outstanding Unclaimed Validator Rewards](#check-outstanding-unclaimed-validator-rewards)
  - [Withdraw Validator Commission](#withdraw-validator-commission)
  - [Check Delegator Rewards](#check-delegator-rewards)
  - [Withdraw All Rewards (Commission + Delegation)](#withdraw-all-rewards-commission--delegation)
  - [Monitor Balance Changes Over Time](#monitor-balance-changes-over-time)
  - [Query Community Pool](#query-community-pool)
  - [Get Comprehensive Distribution Info](#get-comprehensive-distribution-info)
  - [Withdraw All Delegation Rewards](#withdraw-all-delegation-rewards)
- [Quick Reference Playbook](#quick-reference-playbook)
  - [Daily Validator Rewards Check](#daily-validator-rewards-check)
  - [Complete Rewards Withdrawal](#complete-rewards-withdrawal)
  - [Dry Run Before Withdrawal](#dry-run-before-withdrawal)

:::info Read the official Cosmos documentation for more information

Validator tx fees functionality is directly adopted from the Cosmos SDK [x/distribution](https://docs.cosmos.network/main/build/modules/distribution).

Make sure to read those docs as the primary source of truth.

:::

## Identifying the Validator Sets

:::note TODO(@olshansk) Streamline this section

To understand which validator should receive the rewards at a certain block, you
need to cross-reference the `Consensus_Pubkey_ed25519` from section (2) with the `ConsPubKey_Encoded_ed25519` from section (1)
and check the `ConsAddress` corresponding to a certain block in section (3).

:::

### 1. View CometBFT Consensus Validator Set (`ed25519`)

View all active validators with their voting power and proposer priority:

```bash
pocketd query comet-validator-set --network=main -o json \
  | jq -r '["ConsAddress","ConsPubKey_Encoded_ed25519","VotingPower","ProposerPriority"],
           (.validators[] | [.address, .pub_key.key, .voting_power, .proposer_priority])
           | @tsv' \
  | column -t
```

Outputting a table like so:

```bash
ConsAddress                                         ConsPubKey_Encoded_ed25519                    VotingPower  ProposerPriority
poktvalcons15jv09gszged6n4p5yx6cylx574lvdt35fvuyxa  Td0lmPVFbNCXNRvds2HUbOvAW4H9WHf0lvTsLu2bdig=  3998646      4065371
poktvalcons1uyrynx4fylgnnfkzjkfjuknypzczc64drjtysy  MJ/4c5pwP3e8tdbyJEv3TFIlPzVTrJw9sOpkdW/gBY0=  2020000      -2082881
poktvalcons1lxz5u0938e54qx6ut9kpldayfkerrvuwaxff4d  YdlQyhjtrq9pk7afmz6oQ275L4FElzjzEJvB1fj3e1w=  2017247      3979173
poktvalcons1cpn7zhcvklf2kj8tczyhmx6pu5n8qzte6juduy  +laIiedk/ueBkutyOPWrCImH5cFtuR8Hywxvrz5FesQ=  2015000      1308627
poktvalcons1g4mm5lus677m6efvjmvjetw5h3xplu8gy50t3y  jadC4IdoEiRB+nzjp29qiq2mJqj3ZjQ+AELJ6AVgnAM=  2010000      -4143495
poktvalcons1weegm9a5nwe7xjqlfw4wp6wh0le4mljhm0gzey  GfvVW9IMypLv+AdlvvZpiduW1/whjt7vI/9nc7m8Its=  2010000      -3126795
```

### 2. List CosmosSDK Bonded Validator Operator Addresses (`secp256k1`)

Get all bonded (active) validators that are not jailed:

```bash
pocketd query staking validators --output json --network=main --grpc-insecure=false \
  | jq -r '["Operator_Address_secp256k1","Consensus_Pubkey_ed25519"],
           (.validators[] | select(.jailed != true and .status=="BOND_STATUS_BONDED") | [.operator_address, .consensus_pubkey.value])
           | @tsv' | column -t
```

Outputting a table like so:

```bash
Operator_Address_secp256k1                          Consensus_Pubkey_ed25519
poktvaloper1zppmwrdgvywrc66nn2u40ad90na9983fu9yh55  MJ/4c5pwP3e8tdbyJEv3TFIlPzVTrJw9sOpkdW/gBY0=
poktvaloper1zdyjlf9ytwahsaawwym0uzq7z8eu9me8upl2sa  jadC4IdoEiRB+nzjp29qiq2mJqj3ZjQ+AELJ6AVgnAM=
poktvaloper18808wvw0h4t450t06uvauny8lvscsxjfx0wu80  GfvVW9IMypLv+AdlvvZpiduW1/whjt7vI/9nc7m8Its=
poktvaloper1gr3k0kvv4mapg8ev53uuvnquw63yt50wwehys3  Td0lmPVFbNCXNRvds2HUbOvAW4H9WHf0lvTsLu2bdig=
poktvaloper12l2xmehtylf4m2vlddsxcgj3hlx8r6266pcaa3  +laIiedk/ueBkutyOPWrCImH5cFtuR8Hywxvrz5FesQ=
poktvaloper1kmgup24j246n8lpjha32r032d3ps3vc0g6xeff  YdlQyhjtrq9pk7afmz6oQ275L4FElzjzEJvB1fj3e1w=
```

### 3. Check Current Block Proposer (encoded `ed25519`)

Identify the proposer for recent blocks:

```bash
latest_height=$(pocketd status --network=main -o json | jq -r '.sync_info.latest_block_height')
num_blocks=50
for ((h=latest_height; h>latest_height-num_blocks; h--)); do
  proposer_b64=$(pocketd query block --type=height $h --network=main -o json | jq -r '.header.proposer_address')
  proposer_valcons=$(pocketd debug addr $(echo "$proposer_b64" | base64 --decode | xxd -p -c256 | tr -d '\n') \
                       | grep "Bech32 Con:" | awk '{print $3}')
  echo "Block $h: $proposer_valcons"
done
```

Outputting a list like so:

```bash
Block 304457: poktvalcons1lxz5u0938e54qx6ut9kpldayfkerrvuwaxff4d
Block 304456: poktvalcons15jv09gszged6n4p5yx6cylx574lvdt35fvuyxa
Block 304455: poktvalcons15jv09gszged6n4p5yx6cylx574lvdt35fvuyxa
Block 304454: poktvalcons1weegm9a5nwe7xjqlfw4wp6wh0le4mljhm0gzey
Block 304453: poktvalcons1uyrynx4fylgnnfkzjkfjuknypzczc64drjtysy
Block 304452: poktvalcons15jv09gszged6n4p5yx6cylx574lvdt35fvuyxa
Block 304451: poktvalcons1cpn7zhcvklf2kj8tczyhmx6pu5n8qzte6juduy
Block 304450: poktvalcons1lxz5u0938e54qx6ut9kpldayfkerrvuwaxff4d
Block 304449: poktvalcons15jv09gszged6n4p5yx6cylx574lvdt35fvuyxa
Block 304448: poktvalcons1g4mm5lus677m6efvjmvjetw5h3xplu8gy50t3y
```

## Inspecting and Retrieving Block TX Fees

### Check Validator Commission

View accumulated tx commissions that haven't been withdrawn across all validators:

```bash
pocketd query distribution community-pool --network=main --grpc-insecure=false -o json | jq
```

### Using your validator key

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

````bash
# Replace with your validator account address
ACCOUNT_ADDR="pokt18808wvw0h4t450t06uvauny8lvscsxjfyua7vh"

# Get latest block height from mainnet RPC
latest_height=$(curl -s https://shannon-grove-rpc.mainnet.poktroll.com/status | jq -r '.result.sync_info.latest_block_height')

# Check balance every 100 blocks for the last 1000 blocks
for ((h=latest_height-1; h>latest_height-1000; h-=100)); do
  echo -n "Height $h: "
  curl -s -H "x-cosmos-block-height: $h" \
    https://shannon-grove-api.mainnet.poktroll.com/cosmos/bank/v1beta1/balances/$ACCOUNT_ADDR \
    | jq -r '.balances[]? | select(.denom=="upokt") | .amount // "0"'
done

## Inspecting and Retrieving Relay Settlement Fees

### Check Distribution Parameters

Verify the current distribution parameters for relay rewards:

```bash
pocketd query tokenomics params --network=main --grpc-insecure=false -o json | jq
````

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
