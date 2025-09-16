---
title: Validator Reward Playbook
sidebar_position: 2
---

This playbook provides step-by-step instructions for tracking and inspecting validator rewards on Pocket Network.

_**NOTE**: It is the first (terse) version of the instructions and will be simplified into a more user-friendly script in the future._

## Table of Contents <!-- omit in toc -->

- [A. Identifying the Validator Sets](#a-identifying-the-validator-sets)
  - [1. View CometBFT Consensus Validator Set (`ed25519`)](#1-view-cometbft-consensus-validator-set-ed25519)
  - [2. List CosmosSDK Bonded Validator Operator Addresses (`secp256k1`)](#2-list-cosmossdk-bonded-validator-operator-addresses-secp256k1)
  - [3. Check Current Block Proposer (encoded `ed25519`)](#3-check-current-block-proposer-encoded-ed25519)
- [B. Monitoring Validator Balance Over Time](#b-monitoring-validator-balance-over-time)
  - [1. Get the Validator Account Address](#1-get-the-validator-account-address)
  - [2. Monitor Balance Changes Over Time](#2-monitor-balance-changes-over-time)
- [C. Validator Rewards](#c-validator-rewards)
  - [Check Community Pool Commission Accumulated](#check-community-pool-commission-accumulated)
  - [View Validator Commission Accumulated](#view-validator-commission-accumulated)
  - [Withdraw Validator Commission Rewards](#withdraw-validator-commission-rewards)
- [D. \[WIP\] Delegator Rewards](#d-wip-delegator-rewards)
  - [Check Delegator Rewards](#check-delegator-rewards)
  - [Withdraw Delegator Rewards](#withdraw-delegator-rewards)
- [E. \[WIP\] Tokenomics Relay Distribution Parameters](#e-wip-tokenomics-relay-distribution-parameters)

:::info Read the official Cosmos documentation for more information

Validator tx fees functionality is directly adopted from the Cosmos SDK [x/distribution](https://docs.cosmos.network/main/build/modules/distribution).

Make sure to read those docs as the primary source of truth.

:::

## A. Identifying the Validator Sets

:::note TODO(@olshansk) Streamline this whole page

To understand which validator should receive the rewards at a certain block, you
need to cross-reference the `Consensus_Pubkey_ed25519` from section (A2) with the `ConsPubKey_Encoded_ed25519` from section (A1)
and check the `ConsAddress` corresponding to a certain block in section (A3).

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
...
```

### 2. List CosmosSDK Bonded Validator Operator Addresses (`secp256k1`)

Get all bonded (active) validators that are not jailed:

```bash
pocketd query staking validators --output json --network=main \
  | jq -r '["Operator_Address_secp256k1","Consensus_Pubkey_ed25519"],
           (.validators[] | select(.jailed != true and .status=="BOND_STATUS_BONDED") | [.operator_address, .consensus_pubkey.value])
           | @tsv' | column -t
```

Outputting a table like so:

```bash
Operator_Address_secp256k1                          Consensus_Pubkey_ed25519
poktvaloper1zppmwrdgvywrc66nn2u40ad90na9983fu9yh55  MJ/4c5pwP3e8tdbyJEv3TFIlPzVTrJw9sOpkdW/gBY0=
poktvaloper1zdyjlf9ytwahsaawwym0uzq7z8eu9me8upl2sa  jadC4IdoEiRB+nzjp29qiq2mJqj3ZjQ+AELJ6AVgnAM=
...
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
...
Block 304455: poktvalcons15jv09gszged6n4p5yx6cylx574lvdt35fvuyxa
Block 304454: poktvalcons1weegm9a5nwe7xjqlfw4wp6wh0le4mljhm0gzey
...
```

## B. Monitoring Validator Balance Over Time

### 1. Get the Validator Account Address

Retrieve the validator operator address (`poktvaloper1...`) from the section above and run:

```bash
pocketd debug addr poktvaloper18808wvw0h4t450t06uvauny8lvscsxjfx0wu80
```

Which will output:

```bash
Address: ...
Address (hex): ...
Bech32 Acc: pokt18808wvw0h4t450t06uvauny8lvscsxjfyua7vh
Bech32 Val: poktvaloper18808wvw0h4t450t06uvauny8lvscsxjfx0wu80
Bech32 Con: poktvalcons18808wvw0h4t450t06uvauny8lvscsxjfjuaqtw
```

Use the account address (e.g. `pokt18808wvw0h4t450t06uvauny8lvscsxjfyua7vh`) in the next step.

### 2. Monitor Balance Changes Over Time

Track your validator's balance changes across block heights:

```bash
# Replace with your validator account address
ACCOUNT_ADDR="pokt18808wvw0h4t450t06uvauny8lvscsxjfyua7vh"

# Get latest block height from mainnet RPC
latest_height=$(curl -s https://shannon-grove-rpc.mainnet.poktroll.com/status | jq -r '.result.sync_info.latest_block_height')

# Check balance every 100 blocks for the last 1000 blocks
for ((h=latest_height-1000; h<latest_height; h+=100)); do
  echo -n "Height $h: "
  curl -s -H "x-cosmos-block-height: $h" \
    https://shannon-grove-api.mainnet.poktroll.com/cosmos/bank/v1beta1/balances/$ACCOUNT_ADDR \
    | jq -r '.balances[]? | select(.denom=="upokt") | .amount // "0"'
done
```

## C. Validator Rewards

### Check Community Pool Commission Accumulated

View accumulated tx commissions that haven't been withdrawn across all validators:

```bash
pocketd query distribution community-pool --network=main -o json | jq
```

### View Validator Commission Accumulated

```bash
export VALIDATOR_ADDRESS="poktvaloper18808wvw0h4t450t06uvauny8lvscsxjfx0wu80"

# View all un-withdrawn rewards for a particular validator:
echo -e "\n === View all un-withdrawn rewards for a particular validator ==="
pocketd query distribution validator-outstanding-rewards $VALIDATOR_ADDRESS --network=main -o json | jq

# View accumulated tx commissions that haven't been withdrawn across all validators:
echo -e "\n === View accumulated tx commissions that haven't been withdrawn across all validators ==="
pocketd query distribution commission $VALIDATOR_ADDRESS --network=main -o json | jq

# View all distribution-related information for your validator:
echo -e "\n === View all distribution-related information for your validator ==="
pocketd query distribution validator-distribution-info $VALIDATOR_ADDRESS --network=main -o json | jq
```

### Withdraw Validator Commission Rewards

Withdraw rewards from for your validator and its delegations:

```bash
pocketd tx distribution withdraw-all-rewards \
  --from=pokt18808wvw0h4t450t06uvauny8lvscsxjfyua7vh \
  --network=main --gas=auto --fees=10upokt
```

## D. [WIP] Delegator Rewards

### Check Delegator Rewards

View rewards from delegations (including self-delegation):

```bash
pocketd query distribution rewards <delegator-address> --network=main -o json | jq
```

### Withdraw Delegator Rewards

```bash
pocketd tx distribution withdraw-all-rewards --from=<delegator-address> \
  --network=main --gas=auto --fees=10upokt
```

## E. [WIP] Tokenomics Relay Distribution Parameters

<details>
<summary>WIP for viewing tokenomics relay distribution parameters</summary>

Verify the current distribution parameters for relay rewards:

```bash
pocketd query tokenomics params --network=main -o json | jq
```

Key parameters to check:

- `mint_allocation_percentages.proposer`: Proposer's share of new mints
- `mint_equals_burn_claim_distribution.proposer`: Proposer's share when mint equals burn

</details>
