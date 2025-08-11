---
title: Validator Rewards
sidebar_position: 1
---

This guide demonstrates common Vultr API operations for managing virtual machine instances via the [Vultr API](https://www.vultr.com/api).

## Raw Section 1

```markdown
### Checking Accumulated Fees + Block Rewards

Query Outstanding Validator Rewards (Un-withdrawn)

### For a specific validator's outstanding rewards

pocketd query distribution validator-outstanding-rewards <validator-operator-address>

# Example:

pocketd query distribution validator-outstanding-rewards poktvaloper1abc123...

Query Validator Commission

# Check accumulated commission for a validator

pocketd query distribution commission <validator-operator-address>

# Example:

pocketd query distribution commission poktvaloper1abc123...

Query Delegator Rewards

# Check rewards for a delegator (including self-delegation)

pocketd query distribution rewards <delegator-address>

# Check rewards from a specific validator

pocketd query distribution rewards <delegator-address> <validator-operator-address>

Query Community Pool

# Check total fees in community pool

pocketd query distribution community-pool

Query Validator Distribution Info

# Get comprehensive distribution info for a validator

pocketd query distribution validator-distribution-info <validator-operator-address>

2. Withdrawing Fees + Block Rewards

Withdraw Validator Commission

# Withdraw validator commission (must be validator operator)

pocketd tx distribution withdraw-validator-commission \
 --from <validator-operator-key> \
 --chain-id pocket \
 --gas auto \
 --gas-adjustment 1.5 \
 --fees 1000000upokt

Withdraw Delegation Rewards

# Withdraw rewards from a specific validator delegation

pocketd tx distribution withdraw-rewards <validator-operator-address> \
 --from <delegator-key> \
 --chain-id pocket \
 --gas auto \
 --gas-adjustment 1.5 \
 --fees 1000000upokt

# Withdraw ALL delegation rewards at once

pocketd tx distribution withdraw-all-rewards \
 --from <delegator-key> \
 --chain-id pocket \
 --gas auto \
 --gas-adjustment 1.5 \
 --fees 1000000upokt

Combined Withdrawal (Commission + Delegation Rewards)

# If you're withdrawing as a validator operator, you can withdraw both commission and delegation rewards in one command

pocketd tx distribution withdraw-rewards <validator-operator-address> \
 --commission \
 --from <validator-operator-key> \
 --chain-id pocket \
 --gas auto \
 --gas-adjustment 1.5 \
 --fees 1000000upokt

Practical Example Workflow

1. Check if your validator has outstanding rewards:
   pocketd query distribution validator-outstanding-rewards $(pocketd keys show validator --bech val -a)
2. Check your validator's commission:
   pocketd query distribution commission $(pocketd keys show validator --bech val -a)
3. Withdraw both commission and self-delegation rewards:
   pocketd tx distribution withdraw-rewards $(pocketd keys show validator --bech val -a) \
    --commission \
    --from validator \
    --chain-id pocket \
    --gas auto \
    --gas-adjustment 1.5 \
    --fees 1000000upokt

Important Notes

- Outstanding rewards show what's available to withdraw but hasn't been withdrawn yet
- Commission is the validator's cut from delegator rewards
- Withdrawals require gas fees, so ensure you have sufficient balance
- Use --dry-run flag to simulate transactions before executing
- Consider setting a withdrawal address if you want rewards sent to a different account

If the outstanding rewards show zero, it confirms that either no fees are being collected (due to the fee waiver) or they've already been withdrawn.
```

## Raw Section 2

````markdown
Going to use this thread w/ stream-of-throught updates on how I'm testing/evaluating.

## Beta TestNet (after `v0.1.28`) upgrade

1. Check that the gov params are correct ✅
2. Check the comet validator set 🤔 ✅
3. Check the staking validator set ✅
4. Check the staking validator operator set ✅
5. Check the staking validator operator set addresses ✅
6. Look for balance changes ⚪
7. Full Node Logs 🤔 ✅
8. Evaluating the proposer at each block

## 1. Check the gov params

Run this query:

```bash
pocketd query tokenomics params --network=beta -o json | jq
```
````

And ensure `mint_equals_burn_claim_distribution.proposer` != `0`.

```json
{
  "params": {
    "dao_reward_address": "pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t",
    "mint_allocation_percentages": {
      "dao": 0.1,
      "proposer": 0.2,
      "supplier": 0.6,
      "source_owner": 0.1,
      "application": 0
    },
    "global_inflation_per_claim": 0.000001,
    "mint_equals_burn_claim_distribution": {
      "dao": 0.1,
      "proposer": 0.04,
      "supplier": 0.7,
      "source_owner": 0.16,
      "application": 0
    }
  }
}
```

## 2. Check the validator set

```bash
pocketd query comet-validator-set --network=beta -o json | jq
```

Ensure there are non-zero consensus addresses:

```json
{
  "block_height": "225944",
  "validators": [
    {
      "address": "poktvalcons18adykh0m9e5zc4qyue53jaht5ufpgmjh8lzuun",
      "pub_key": {
        "@type": "/cosmos.crypto.ed25519.PubKey",
        "key": "pboqSkgWggAsEa1zFDTAaXrPabIaSevuTCsAFNXZOqQ="
      },
      "voting_power": "1000002000",
      "proposer_priority": "-1816197684"
    },
    {
      "address": "poktvalcons1fvf53nmvadvszxj6w7jpnk56v93ctd4tjuq0r9",
      "pub_key": {
        "@type": "/cosmos.crypto.ed25519.PubKey",
        "key": "Pqa9r5VRXSWh4AGqKXsdtc1hxBsHSspIgftKJIrJl/o="
      },
      "voting_power": "1000002000",
      "proposer_priority": "1625028250"
    },
    {
      "address": "poktvalcons1vdgkzlmrnd586x5h3szww69zqjnyeqvhe60k95",
      "pub_key": {
        "@type": "/cosmos.crypto.ed25519.PubKey",
        "key": "edXdlBiEZSB44ytKUfb9mQoNZ6gilKnO4GoIiUIg5Fc="
      },
      "voting_power": "1000002000",
      "proposer_priority": "-596496983"
    },
    {
      "address": "poktvalcons1lxz5u0938e54qx6ut9kpldayfkerrvuwaxff4d",
      "pub_key": {
        "@type": "/cosmos.crypto.ed25519.PubKey",
        "key": "YdlQyhjtrq9pk7afmz6oQ275L4FElzjzEJvB1fj3e1w="
      },
      "voting_power": "1000002000",
      "proposer_priority": "787666418"
    }
  ],
  "pagination": {
    "next_key": null,
    "total": "4"
  }
}
```

## 4. Check the staking validator operator set

```bash
pocketd query staking validators --output json --network=beta | jq -r '.validators[] | select(.jailed != true and .status=="BOND_STATUS_BONDED") | .operator_address'
```

Ensure we have non-jailed and bonded operators:

```bash
poktvaloper18rdpjl3ndma372h4503ug8cpd6kzwr8hf2799u
poktvaloper1sadgp3998r2wxka9ec88hm0znxumr4m7vyx8dv
poktvaloper14kgy84sgl6a6e4kk8hn3e8gj3c3spk6jvyez3d
poktvaloper1u994thderhxj60pwhjscne8wcfn8efc8veugy2
```

## 5. Check the staking validator operator set addresses

Run `pocket addr poktvaloper1...` for each of the `valoperar` addresses above and:

```
pokt18rdpjl3ndma372h4503ug8cpd6kzwr8hted8wy
pokt1sadgp3998r2wxka9ec88hm0znxumr4m7wh49x5
pokt14kgy84sgl6a6e4kk8hn3e8gj3c3spk6jwh2q64
pokt1u994thderhxj60pwhjscne8wcfn8efc8w2020j
```

## 6. Look for balance changes

```bash
for ((h=223510; h<=225610; h+=100)); do
  echo -n "Height $h: "
  curl -s -H "x-cosmos-block-height: $h" \
    https://shannon-testnet-grove-api.beta.poktroll.com/cosmos/bank/v1beta1/balances/pokt1... \
    | jq -r '.balances[] | select(.denom=="upokt") | .amount // "0"'
done
```

## 7. Full Node Logs

Validate WHO is getting the x/tokenomics rewards via the x/tokenomics module:

We've evaluated full node logs and observed that the proposer distribution is being send to `pokt18rdpjl3ndma372h4503ug8cpd6kzwr8hted8wy`.

<img width="800" height="491" alt="Screenshot 2025-08-11 at 3 06 48 PM" src="https://github.com/user-attachments/assets/7d7ea288-76c2-4029-a736-57a996a1b967" />
<img width="965" height="1229" alt="Screenshot 2025-08-11 at 3 06 43 PM" src="https://github.com/user-attachments/assets/02c40a48-7c13-4dad-88f7-acee5c891808" />

## 8. Proposer at each block

```
 latest=$(pocketd status --network=beta --home=~/.pocket_prod -o json | jq -r '.sync_info.latest_block_height')

for ((h=latest; h>latest-100; h--)); do
  proposer=$(pocketd query block --type=height $h --network=beta --home=~/.pocket_prod -o json \
    | jq -r '.header.proposer_address')
  echo "$h $proposer"
done
```

```bash
226017 SxNIz2zrWQEaWnekGdqaYWOFtqs=
226016 P1pLXfsuaCxUBOZpGXbrpxIUblc=
226015 Y1Fhf2ObaH0al4wE52iiBKZMgZc=
226014 +YVOPLE+aVAbXFlsH7ekTbIxs44=
226013 SxNIz2zrWQEaWnekGdqaYWOFtqs=
226012 P1pLXfsuaCxUBOZpGXbrpxIUblc=
226011 Y1Fhf2ObaH0al4wE52iiBKZMgZc=
226010 +YVOPLE+aVAbXFlsH7ekTbIxs44=
226009 SxNIz2zrWQEaWnekGdqaYWOFtqs=
226008 P1pLXfsuaCxUBOZpGXbrpxIUblc=
226007 Y1Fhf2ObaH0al4wE52iiBKZMgZc=
226006 +YVOPLE+aVAbXFlsH7ekTbIxs44=
226005 SxNIz2zrWQEaWnekGdqaYWOFtqs=
226004 P1pLXfsuaCxUBOZpGXbrpxIUblc=
226003 Y1Fhf2ObaH0al4wE52iiBKZMgZc=
226002 +YVOPLE+aVAbXFlsH7ekTbIxs44=
226001 SxNIz2zrWQEaWnekGdqaYWOFtqs=
226000 P1pLXfsuaCxUBOZpGXbrpxIUblc=
```

## 9. Seeing the balance of a proposer change

```bash
for ((h=223510; h<=225710; h+=100)); do
  echo -n "Height $h: "
  curl -s -H "x-cosmos-block-height: $h" \
    https://shannon-testnet-grove-api.beta.poktroll.com/cosmos/bank/v1beta1/balances/pokt18rdpjl3ndma372h4503ug8cpd6kzwr8hted8wy \
    | jq -r '.balances[] | select(.denom=="upokt") | .amount // "0"'
done
```

```bash
Height 223510: 997518
Height 223610: 997518
Height 223710: 997518
Height 223810: 997518
Height 223910: 997518
Height 224010: 997518
Height 224110: 997518
Height 224210: 997518
Height 224310: 997518
Height 224410: 997518
Height 224510: 997518
Height 224610: 997518
Height 224710: 1025297
Height 224810: 1088841
Height 224910: 1119428
Height 225010: 1174080
Height 225110: 1235499
Height 225210: 1265564
Height 225310: 1328574
Height 225410: 1391003
Height 225510: 1407126
Height 225610: 1470688
Height 225710: 1512825
```

## 10. New Issues

1. Need to enable reward delegation
2. Need to document how to retrieve delegated rewards
3. Need to document how to retrieve fees

```

```
