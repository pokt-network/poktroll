---
title: Tokenomics Inspection Cheatsheet
sidebar_position: 7
---

- [Bash RC helpers](#bash-rc-helpers)
- [`pocketd` CLI queries](#pocketd-cli-queries)
- [Available Onchain Events](#available-onchain-events)
- [Validator Rewards Queries](#validator-rewards-queries)
- [Viewing Account Balance over time](#viewing-account-balance-over-time)
- [Inspecting Claim settlement in a specific block](#inspecting-claim-settlement-in-a-specific-block)

## Bash RC helpers

There are a handful of useful bash helpers in the `tools/rc_helpers` directory.

```bash
source ./tools/rc_helpers/queries.sh
```

```text
Available commands:
  shannon_query_unique_tx_msgs_and_events  - Get unique message and event types
  shannon_query_unique_block_events        - Get unique block events
  ...
```

## `pocketd` CLI queries

To inspect various tokenomics params, you can query the `tokenomics` module

```bash
pocketd q tokenomics --help
```

To inspect onchain claims & proofs, you can query the `proofs` module

```bash
pocketd q proofs --help
```

## Available Onchain Events

You can find all available events by running

```bash
grep -r "message Event" ./proto/pocket/tokenomics
```

And filtering for the events you're interested in inspecting either onchain `txs`:

```bash
pocketd q txs --help
```

Or onchain `blocks`:

```bash
pocketd q block-results --help
```

## Validator Rewards Queries

### Validator Reward Distribution Parameters

Check the current tokenomics parameters that control validator reward distribution:

```bash
# View all tokenomics parameters
pocketd query tokenomics params --network <network>

# Get specific validator allocation percentage
pocketd query tokenomics params --network <network> -o json | jq -r '.mint_allocation_percentages.proposer'

# Get global inflation per claim
pocketd query tokenomics params --network <network> -o json | jq -r '.global_inflation_per_claim'
```

### Validator and Delegator Account Balance Queries

Check validator and delegator account balances (where rewards are sent directly):

```bash
# Check validator account balance (where validator commission is sent directly)
pocketd query bank balance <validator-account-address> upokt --network <network>

# Check delegator account balance (where delegator rewards are sent directly)  
pocketd query bank balance <delegator-account-address> upokt --network <network>

# Check validator commission rate (for reward calculation reference)
pocketd query staking validator <validator-operator-address> --network <network> | jq '.commission.commission_rates'

# Check delegation shares (for reward proportion calculations)
pocketd query staking delegation <delegator-address> <validator-operator-address> --network <network>

# List all delegations to a validator (to understand total reward distribution)
pocketd query staking delegations-to <validator-operator-address> --network <network>
```

### Staking Information for Reward Distribution

Check validator stakes that determine reward distribution:

```bash
# View the current block proposer and their bonded tokens
pocketd query staking validators --network <network> -o json | jq -r '.validators[] | "\(.operator_address) \(.tokens)"'

# Calculate total bonded tokens
pocketd query staking validators --network <network> -o json | jq -r '.validators | map(.tokens | tonumber) | add'

# Check specific validator staking info
pocketd query staking validator <validator-address> --network <network>
```

### Validator Reward Events

Monitor validator reward distribution through events:

```bash
# Search for claim settlement events
pocketd query txs --events 'pocket.tokenomics.EventClaimSettled.num_relays>0' --network <network>

# Search for validator reward allocation events
pocketd query txs --events 'cosmos.distribution.v1beta1.EventAllocateTokens' --network <network>

# Search for tokenomics module events
pocketd query txs --events 'message.module=tokenomics' --network <network>
```

### Helper Scripts for Validator Reward Analysis

First, source the query helper functions:

```bash
source ./tools/rc_helpers/queries.sh
```

Calculate expected validator reward share:

```bash
# Calculate validator's share of total bonded tokens (determines reward distribution)
shannon_query_validator_reward_share <validator-address> <network>

# Example:
shannon_query_validator_reward_share poktvaloper1abc123... main
```

Monitor validator rewards over time:

```bash
# Monitor validator outstanding rewards with periodic updates  
shannon_monitor_validator_rewards <validator-address> <network> [interval-seconds]

# Example (check every 60 seconds):
shannon_monitor_validator_rewards poktvaloper1abc123... main 60
```

Check recent validator reward settlements:

```bash
# Check recent tokenomics claim settlements that trigger validator rewards
shannon_check_recent_settlements <network> [limit]

# Example:
shannon_check_recent_settlements main 10
```

## Viewing Account Balance over time

The following is an example of how to view the balance of `pokt1lla0yhjf2fhzrlgu6le3ymw9aqayepxlx3lf4q` grow from height `205950` to `210000` in 30 height increments.

```bash
for ((h=205950; h<=210000; h+=30)); do echo -n "Height $h: "; curl -s -H "x-cosmos-block-height: $h" https://shannon-grove-api.mainnet.poktroll.com/cosmos/bank/v1beta1/balances/pokt1lla0yhjf2fhzrlgu6le3ymw9aqayepxlx3lf4q | jq -r '.balances[0].amount // "0"'; done
```

## Inspecting Claim settlement in a specific block

If some block was used for claim settlemend (e.g. `210033`), you can download it like so:

```bash
pocketd query block-results 210033 --network=main --grpc-insecure=false -o json >> block_210033.json
```

And identify all the events related to token transfers associated with a particular account (e.g. `pokt1lla0yhjf2fhzrlgu6le3ymw9aqayepxlx3lf4q`)

```bash
cat block_210033.json | jq -r '.finalize_block_events[]
  | select(.type == "transfer")
  | select(.attributes[]?
      | select(.key == "recipient" and .value == "pokt1lla0yhjf2fhzrlgu6le3ymw9aqayepxlx3lf4q"))
  | .attributes[]
  | select(.key == "amount")
  | .value'
```
