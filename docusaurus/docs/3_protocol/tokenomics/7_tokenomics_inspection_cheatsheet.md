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
  shannon_query_tx_messages                - Query transactions by message type
  shannon_query_tx_events                  - Query transactions by event type
  shannon_query_block_events               - Query block events
  shannon_query_unique_claim_suppliers     - Get unique claim supplier addresses
  shannon_query_supplier_tx_events         - Get supplier-specific transaction events
  shannon_query_supplier_block_events      - Get supplier-specific block events
  shannon_query_application_block_events   - Get application-specific block events
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

### Distribution Module Queries

Check validator rewards in the distribution module:

```bash
# Check distribution module balance (where validator rewards are sent)
pocketd query bank balance cosmos1jv65s3grqf6v6jl3dp4t6c9t9rk99cd88lyufl upokt --network <network>

# View validator outstanding rewards
pocketd query distribution validator-outstanding-rewards <validator-address> --network <network>

# View delegator rewards from specific validator
pocketd query distribution rewards <delegator-address> <validator-address> --network <network>

# View all delegator rewards
pocketd query distribution rewards <delegator-address> --network <network>

# Check validator commission
pocketd query distribution commission <validator-address> --network <network>
```

### Staking Information for Reward Distribution

Check validator stakes that determine reward distribution:

```bash
# View all validators and their bonded tokens
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

Calculate expected validator reward share:

```bash
#!/bin/bash
VALIDATOR_ADDR="<validator-address>"
NETWORK="<network>"

# Get validator bonded tokens
VAL_TOKENS=$(pocketd query staking validator $VALIDATOR_ADDR --network $NETWORK -o json | jq -r '.tokens')

# Get total bonded tokens across all validators  
TOTAL_TOKENS=$(pocketd query staking validators --network $NETWORK -o json | jq -r '.validators | map(.tokens | tonumber) | add')

# Calculate percentage
PERCENTAGE=$(echo "scale=6; $VAL_TOKENS * 100 / $TOTAL_TOKENS" | bc)
echo "Validator $VALIDATOR_ADDR holds $PERCENTAGE% of total bonded tokens"
```

Monitor validator rewards over time:

```bash
#!/bin/bash
VALIDATOR_ADDR="<validator-address>"
NETWORK="<network>"

while true; do
  REWARDS=$(pocketd query distribution validator-outstanding-rewards $VALIDATOR_ADDR --network $NETWORK -o json | jq -r '.rewards[0].amount // "0"')
  echo "$(date): $REWARDS uPOKT outstanding rewards"
  sleep 60
done
```

Check recent validator reward settlements:

```bash
# Get recent tokenomics settlement transactions
pocketd query txs --events 'message.module=tokenomics' --limit 10 --network <network> -o json | \
  jq -r '.txs[] | select(.logs[].events[].type == "pocket.tokenomics.EventClaimSettled") | 
    "\(.timestamp) - Settlement: \(.logs[].events[] | select(.type == "pocket.tokenomics.EventClaimSettled") | 
    .attributes[] | select(.key == "num_relays") | .value) relays"'
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
