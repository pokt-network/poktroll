---
title: Validator Rewards Playbook
sidebar_position: 2
---

# Validator Rewards Playbook

This playbook provides comprehensive guidance for monitoring, inspecting, and understanding validator rewards in the Pocket Network Shannon protocol.

:::tip What You'll Learn

- How validator rewards are distributed during session settlement
- How to retrieve and inspect validator rewards and delegation rewards  
- How to monitor reward distribution using telemetry and queries
- How to validate that rewards are being distributed correctly
- Troubleshooting common validator reward issues

:::

## Overview

In Pocket Network's Shannon protocol, the block proposer receives rewards as part of the session settlement process through the **Token Logic Modules (TLMs)**. Both the **Global Mint TLM** and **RelayBurnEqualsMint TLM** distribute a configurable percentage of settlement rewards to the current block proposer.

### Key Concepts

- **Proposer Rewards**: Distributed to the current block proposer only
- **Direct Distribution**: Rewards are sent directly to validator and delegator accounts immediately
- **Commission**: Validators earn commission on rewards before distributing to delegators
- **Delegator Rewards**: Distributed directly to individual delegator accounts automatically
- **Settlement-Based**: Rewards are generated during relay session settlement, not block validation

## Validator Reward Distribution Mechanics

### How Rewards Flow

1. **Session Settlement**: When a supplier's claim is settled, both TLMs may distribute validator rewards
2. **Proposer Allocation**: A percentage (configurable via `proposer` parameter in each TLM) goes to the block proposer
3. **Proposer Distribution**: Rewards are distributed to the current block proposer only
4. **Direct Distribution**: 
   - **Validator Commission**: Calculated based on validator's commission rate and sent directly to validator account
   - **Delegator Rewards**: Remaining rewards distributed directly to individual delegator accounts based on their delegation shares
5. **Immediate Settlement**: All distributions happen automatically during session settlement

### Reward Calculation Formula

```
Total Session Settlement = Relays × CUPR × Multiplier  
Global Inflation = Settlement × GlobalInflationPerClaim
Proposer Rewards = (Settlement + Inflation) × ProposerAllocation
```

### Example Calculation

For a session with 20 relays, 100 CUPR, and 42 multiplier:
- Settlement: `20 × 100 × 42 = 84,000 uPOKT`
- Global Inflation (10%): `84,000 × 0.1 = 8,400 uPOKT`
- Proposer Rewards (10%): `(84,000 + 8,400) × 0.1 = 9,240 uPOKT`

**Proposer-Only Distribution:** The full 9,240 uPOKT goes to the current block proposer and their delegators:

Assuming the block proposer has 10% commission and 3 delegators with the following stakes:
- Delegator 1: 5,000,000 uPOKT (62.5% of validator's total)
- Delegator 2: 2,000,000 uPOKT (25% of validator's total) 
- Delegator 3: 1,000,000 uPOKT (12.5% of validator's total)

Distribution:
- **Validator Commission (10%)**: `9,240 × 0.1 = 924 uPOKT` → Block proposer's account
- **Delegator Pool**: `9,240 - 924 = 8,316 uPOKT` distributed proportionally:
  - Delegator 1: `8,316 × 0.625 = 5,198 uPOKT`
  - Delegator 2: `8,316 × 0.25 = 2,079 uPOKT` 
  - Delegator 3: `8,316 × 0.125 = 1,040 uPOKT` (includes remainder)

## Querying Validator Rewards

### Check Tokenomics Parameters

View current reward distribution parameters:

```bash
pocketd query tokenomics params --network <network>
```

Key parameters to monitor:
- `global_inflation_per_claim`: Controls total inflation per settlement
- `mint_allocation_percentages.proposer`: Percentage going to validators
- `dao_reward_address`: Where remaining rewards go

### Check Validator Account Balance

View validator account balance (where validator commission is sent directly):

```bash
pocketd query bank balance <validator-account-address> upokt --network <network>
```

### Check Delegator Account Balances

View delegator account balance (where delegator rewards are sent directly):

```bash
pocketd query bank balance <delegator-account-address> upokt --network <network>
```

### Check Validator Commission Settings

View validator commission rate (used for reward distribution calculations):

```bash
pocketd query staking validator <validator-operator-address> --network <network>
```

Look for the `commission.commission_rates` section to see the current commission rate.

### Check Delegation Information

View delegation details to understand reward distribution:

```bash
pocketd query staking delegations-to <validator-operator-address> --network <network>
```

## Monitoring Validator Rewards

### Real-Time Monitoring

#### 1. Watch Settlement Events

Monitor for claim settlement events that trigger validator rewards:

```bash
pocketd query tx --hash <tx-hash> --network <network>
```

Look for events:
- `pocket.tokenomics.EventClaimSettled` - Main settlement event that triggers validator rewards

#### 2. Monitor Validator Account Balance

Track validator account balance changes over time:

```bash
#!/bin/bash
VALIDATOR_ACCOUNT="<your-validator-account-address>"
NETWORK="<your-network>"

while true; do
  BALANCE=$(pocketd query bank balance $VALIDATOR_ACCOUNT upokt --network $NETWORK --output json | jq -r '.balance.amount')
  echo "$(date): Validator balance: $BALANCE uPOKT"
  sleep 30
done
```

#### 3. Monitor Delegator Account Balances

Track delegator account balance changes:

```bash
#!/bin/bash
DELEGATOR_ACCOUNT="<delegator-account-address>"
NETWORK="<your-network>"

while true; do
  BALANCE=$(pocketd query bank balance $DELEGATOR_ACCOUNT upokt --network $NETWORK --output json | jq -r '.balance.amount')
  echo "$(date): Delegator balance: $BALANCE uPOKT"
  sleep 60
done
```

### Telemetry Metrics

If telemetry is enabled, monitor these metrics:

- `tokenomics.tlm_global_mint.validator_reward_distribution`: Individual validator reward amounts
- `tokenomics.tlm_global_mint.total_validator_rewards`: Total validator rewards per settlement

## Validating Reward Distribution

### Expected Behavior Checklist

✅ **Rewards are distributed to the block proposer only**  
✅ **Delegator rewards are proportional to delegation shares within the proposer's validator**  
✅ **No rewards are lost to rounding errors** (remainder given to validator as additional commission)  
✅ **Non-proposer validators receive zero rewards**  
✅ **Delegators receive rewards minus validator commission**  
✅ **Account balances increase immediately** after settlements
✅ **Commission is calculated correctly** based on validator commission rates  

### Validation Scripts

#### Verify Block Proposer Reward Distribution

```bash
#!/bin/bash
NETWORK="<your-network>"

# Get the current block proposer from the latest block
LATEST_HEIGHT=$(pocketd query block --network $NETWORK | jq -r '.block.header.height')
PROPOSER_ADDRESS=$(pocketd query block $LATEST_HEIGHT --network $NETWORK | jq -r '.block.header.proposer_address')

echo "Latest Block Height: $LATEST_HEIGHT"
echo "Block Proposer: $PROPOSER_ADDRESS"

# Convert consensus address to validator operator address
VALIDATOR_INFO=$(pocketd query staking validators --network $NETWORK --output json | jq -r --arg addr "$PROPOSER_ADDRESS" '.validators[] | select(.consensus_pubkey.value == $addr) | "\(.operator_address) \(.tokens) \(.commission.commission_rates.rate)"')

if [ -n "$VALIDATOR_INFO" ]; then
    read -r operator_addr tokens commission_rate <<< "$VALIDATOR_INFO"
    echo "Proposer Validator: $operator_addr"
    echo "Validator Stake: $tokens tokens"
    echo "Commission Rate: $commission_rate"
    
    # Get delegations for this validator
    echo "Delegations to block proposer:"
    pocketd query staking delegations-to $operator_addr --network $NETWORK --output json | jq -r '.delegation_responses[] | "\(.delegation.delegator_address): \(.delegation.shares)"'
else
    echo "Could not find validator information for proposer"
fi
```

#### Monitor Delegation Rewards

```bash
#!/bin/bash
DELEGATOR_ADDR="<delegator-account-address>"
VALIDATOR_ADDR="<validator-operator-address>" 
NETWORK="<your-network>"

# Check delegation details
DELEGATION_INFO=$(pocketd query staking delegation $DELEGATOR_ADDR $VALIDATOR_ADDR --network $NETWORK --output json)
DELEGATION_SHARES=$(echo $DELEGATION_INFO | jq -r '.delegation.shares')

echo "Delegation shares: $DELEGATION_SHARES"

# Monitor balance changes (rewards are sent directly to delegator account)
INITIAL_BALANCE=$(pocketd query bank balance $DELEGATOR_ADDR upokt --network $NETWORK --output json | jq -r '.balance.amount')
echo "Initial delegator balance: $INITIAL_BALANCE uPOKT"

# Wait for settlement events and check balance changes
sleep 60

UPDATED_BALANCE=$(pocketd query bank balance $DELEGATOR_ADDR upokt --network $NETWORK --output json | jq -r '.balance.amount')
REWARD_INCREASE=$((UPDATED_BALANCE - INITIAL_BALANCE))

echo "Updated delegator balance: $UPDATED_BALANCE uPOKT"
echo "Reward increase: $REWARD_INCREASE uPOKT"
```

## Troubleshooting

### Common Issues

#### 1. No Validator Rewards Being Distributed

**Symptoms**: Validator and delegator account balances not increasing after claim settlements

**Possible Causes**:
- `mint_allocation_percentages.proposer` is set to 0 in tokenomics params
- `mint_equals_burn_claim_distribution.proposer` is set to 0 in tokenomics params  
- No claim settlements occurring (no relay traffic)
- The block proposer has zero bonded stake

**Diagnostics**:
```bash
# Check tokenomics parameters (both proposer percentages)
pocketd query tokenomics params --network <network>

# Check for recent settlements
pocketd query tx --events 'pocket.tokenomics.EventClaimSettled.num_relays>0' --network <network>

# Check validator stakes  
pocketd query staking validators --network <network>

# Look for account balance changes from recent settlements
pocketd query tx --events 'message.module=tokenomics' --network <network>
```

#### 2. Uneven Reward Distribution

**Symptoms**: Validators receiving rewards that don't match their staking weight

**Possible Causes**:
- Recent validator stake changes
- Validator bonding/unbonding events
- Calculation errors (should not happen with proper implementation)

**Diagnostics**:
```bash
# Check validator bonding status
pocketd query staking validator <validator-addr> --network <network>

# Check recent staking transactions
pocketd query tx --events 'message.module=staking' --network <network>
```

#### 3. Delegators Not Receiving Rewards

**Symptoms**: Delegator account balance not increasing despite validator receiving commission

**Possible Causes**:
- High validator commission (100%)
- Recent delegation changes
- Small delegations resulting in negligible rewards
- Validator has zero or invalid delegator shares

**Diagnostics**:
```bash
# Check validator commission rate
pocketd query staking validator <validator-operator-addr> --network <network>

# Check delegation details and shares
pocketd query staking delegation <delegator-addr> <validator-operator-addr> --network <network>

# Check all delegations to validator
pocketd query staking delegations-to <validator-operator-addr> --network <network>

# Monitor account balance changes during settlement
pocketd query bank balance <delegator-account-addr> upokt --network <network>
```

### Debug Commands

#### View Recent Tokenomics Events

```bash
pocketd query tx --events 'message.module=tokenomics' --limit 20 --network <network>
```

#### Inspect Specific Settlement Transaction

```bash
pocketd query tx <settlement-tx-hash> --network <network> --output json | jq '.events[] | select(.type | contains("EventClaimSettled"))'
```

#### Check Module Account Balances

```bash
# Tokenomics module (source of reward transfers)
pocketd query bank balance cosmos1jv65s3grqf6v6jl3dp4t6c9t9rk99cd8w8zk8h upokt --network <network>

# Check individual validator and delegator account balances
pocketd query bank balance <validator-account-address> upokt --network <network>
pocketd query bank balance <delegator-account-address> upokt --network <network>
```

## Best Practices for Validators

### For Validator Operators

1. **Monitor Commission Settings**: Balance earning commission with attracting delegators
2. **Track Outstanding Rewards**: Regularly check accumulated rewards
3. **Watch Settlement Activity**: Monitor relay traffic and claim settlements in your network
4. **Set Up Alerting**: Monitor unusual reward distribution patterns

### For Delegators  

1. **Compare Commission Rates**: Choose validators with reasonable commission rates
2. **Monitor Account Balance**: Track your account balance changes to see direct reward deposits
3. **Automatic Distribution**: Rewards are sent directly to your account (no withdrawal needed)
4. **Diversify Delegations**: Consider spreading stake across multiple validators
5. **Account Security**: Secure your account private keys as rewards go directly there

### Sample Monitoring Script

```bash
#!/bin/bash

# Configuration
VALIDATOR_ACCOUNT="<your-validator-account-address>"
DELEGATOR_ACCOUNT="<your-delegator-account-address>"  
NETWORK="<your-network>"
ALERT_THRESHOLD=1000000  # Alert if balance increases exceed 1M uPOKT

# Get initial balances
INITIAL_VAL_BALANCE=$(pocketd query bank balance $VALIDATOR_ACCOUNT upokt --network $NETWORK --output json | jq -r '.balance.amount')
INITIAL_DEL_BALANCE=$(pocketd query bank balance $DELEGATOR_ACCOUNT upokt --network $NETWORK --output json | jq -r '.balance.amount')

while true; do
    # Check current account balances
    CURRENT_VAL_BALANCE=$(pocketd query bank balance $VALIDATOR_ACCOUNT upokt --network $NETWORK --output json | jq -r '.balance.amount')
    CURRENT_DEL_BALANCE=$(pocketd query bank balance $DELEGATOR_ACCOUNT upokt --network $NETWORK --output json | jq -r '.balance.amount')
    
    # Calculate increases since start
    VAL_INCREASE=$((CURRENT_VAL_BALANCE - INITIAL_VAL_BALANCE))
    DEL_INCREASE=$((CURRENT_DEL_BALANCE - INITIAL_DEL_BALANCE))
    
    echo "$(date):"
    echo "  Validator Balance: $CURRENT_VAL_BALANCE uPOKT (+$VAL_INCREASE)"
    echo "  Delegator Balance: $CURRENT_DEL_BALANCE uPOKT (+$DEL_INCREASE)"
    
    # Alert if balance increases are high
    if [ "$DEL_INCREASE" -gt "$ALERT_THRESHOLD" ]; then
        echo "🚨 ALERT: Delegator balance increased by $DEL_INCREASE uPOKT since monitoring started!"
    fi
    
    sleep 300  # Check every 5 minutes
done
```

This playbook provides the foundation for monitoring and understanding validator rewards in Pocket Network. For additional support, consult the network documentation or community channels.