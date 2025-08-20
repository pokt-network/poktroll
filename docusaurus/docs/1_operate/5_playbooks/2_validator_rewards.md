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

In Pocket Network's Shannon protocol, validators receive rewards as part of the session settlement process through the **Token Logic Modules (TLMs)**. Specifically, the **Global Mint TLM** distributes a configurable percentage of settlement rewards to validators based on their staking weight.

### Key Concepts

- **Validator Rewards**: Distributed proportionally to all bonded validators based on staking weight
- **Delegator Rewards**: Distributed to delegators through Cosmos SDK's distribution module  
- **Commission**: Validators earn commission on rewards before distributing to delegators
- **Settlement-Based**: Rewards are generated during relay session settlement, not block validation

## Validator Reward Distribution Mechanics

### How Rewards Flow

1. **Session Settlement**: When a supplier's claim is settled, the Global Mint TLM calculates inflation
2. **Validator Allocation**: A percentage (configurable via `proposer` parameter) goes to validators  
3. **Stake-Weight Distribution**: Rewards are distributed to ALL validators based on their bonded stake
4. **Delegator Distribution**: The Cosmos SDK distribution module handles delegator rewards and commission

### Reward Calculation Formula

```
Total Session Settlement = Relays × CUPR × Multiplier  
Global Inflation = Settlement × GlobalInflationPerClaim
Validator Rewards = (Settlement + Inflation) × ProposerAllocation
Individual Validator Reward = Validator Rewards × (ValidatorStake / TotalBondedStake)
```

### Example Calculation

For a session with 20 relays, 100 CUPR, and 42 multiplier:
- Settlement: `20 × 100 × 42 = 84,000 uPOKT`
- Global Inflation (10%): `84,000 × 0.1 = 8,400 uPOKT`
- Validator Rewards (10%): `(84,000 + 8,400) × 0.1 = 9,240 uPOKT`

If there are 3 validators with stakes of 700K, 200K, and 100K tokens:
- Validator 1: `9,240 × (700,000 / 1,000,000) = 6,468 uPOKT`
- Validator 2: `9,240 × (200,000 / 1,000,000) = 1,848 uPOKT`
- Validator 3: `9,240 × (100,000 / 1,000,000) = 924 uPOKT`

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

### Check Distribution Module State

View the distribution module balance (where validator rewards are sent):

```bash
pocketd query bank balance cosmos1jv65s3grqf6v6jl3dp4t6c9t9rk99cd88lyufl upokt --network <network>
```

### Check Validator Outstanding Rewards  

View accumulated rewards for a specific validator:

```bash
pocketd query distribution validator-outstanding-rewards <validator-address> --network <network>
```

### Check Delegator Rewards

View pending rewards for a delegator from a specific validator:

```bash
pocketd query distribution rewards-by-validator <delegator-address> <validator-address> --network <network>
```

View all delegator rewards:

```bash
pocketd query distribution rewards <delegator-address> --network <network>
```

### Check Validator Commission

View validator commission settings and accumulated commission:

```bash
pocketd query distribution commission <validator-address> --network <network>
```

## Monitoring Validator Rewards

### Real-Time Monitoring

#### 1. Watch Settlement Events

Monitor for claim settlement events that trigger validator rewards:

```bash
pocketd query tx --hash <tx-hash> --network <network>
```

Look for events:
- `pocket.tokenomics.EventClaimSettled`
- `cosmos.distribution.v1beta1.EventAllocateTokens`

#### 2. Monitor Distribution Module Balance

Track the distribution module balance over time:

```bash
while true; do
  echo "$(date): $(pocketd query bank balance cosmos1jv65s3grqf6v6jl3dp4t6c9t9rk99cd88lyufl upokt --network <network> --output json | jq -r '.balance.amount')"
  sleep 30
done
```

#### 3. Monitor Validator Rewards Accumulation

Track validator outstanding rewards:

```bash
#!/bin/bash
VALIDATOR_ADDR="<your-validator-address>"
NETWORK="<your-network>"

while true; do
  REWARDS=$(pocketd query distribution validator-outstanding-rewards $VALIDATOR_ADDR --network $NETWORK --output json | jq -r '.rewards[0].amount // "0"')
  echo "$(date): Validator rewards: $REWARDS uPOKT"
  sleep 60
done
```

### Telemetry Metrics

If telemetry is enabled, monitor these metrics:

- `tokenomics.tlm_global_mint.validator_reward_distribution`: Individual validator reward amounts
- `tokenomics.tlm_global_mint.total_validator_rewards`: Total validator rewards per settlement

## Validating Reward Distribution

### Expected Behavior Checklist

✅ **Rewards are distributed to ALL validators** (not just block proposer)  
✅ **Distribution is proportional to staking weight**  
✅ **No rewards are lost to rounding errors** (last validator gets remainder)  
✅ **Validators with zero stake receive zero rewards**  
✅ **Delegators receive rewards minus validator commission**  
✅ **Distribution module balance increases after settlements**  

### Validation Scripts

#### Verify Proportional Distribution

```bash
#!/bin/bash
NETWORK="<your-network>"

# Get all validators and their stakes
VALIDATORS=$(pocketd query staking validators --network $NETWORK --output json | jq -r '.validators[] | "\(.operator_address) \(.tokens)"')

echo "Validator Stakes:"
TOTAL_STAKE=0
while read -r validator_addr stake; do
    echo "$validator_addr: $stake tokens"
    TOTAL_STAKE=$((TOTAL_STAKE + stake))
done <<< "$VALIDATORS"

echo "Total Stake: $TOTAL_STAKE"

# Calculate expected distribution for recent rewards
echo "Expected reward distribution (proportional to stake):"
while read -r validator_addr stake; do
    PERCENTAGE=$(echo "scale=4; $stake * 100 / $TOTAL_STAKE" | bc)
    echo "$validator_addr: $PERCENTAGE% of rewards"
done <<< "$VALIDATORS"
```

#### Check Delegation Rewards

```bash
#!/bin/bash
DELEGATOR_ADDR="<delegator-address>"
VALIDATOR_ADDR="<validator-address>" 
NETWORK="<your-network>"

# Check current rewards
CURRENT_REWARDS=$(pocketd query distribution rewards-by-validator $DELEGATOR_ADDR $VALIDATOR_ADDR --network $NETWORK --output json | jq -r '.rewards[0].amount // "0"')

echo "Current pending rewards: $CURRENT_REWARDS uPOKT"

# Withdraw rewards
echo "Withdrawing rewards..."
pocketd tx distribution withdraw-rewards $VALIDATOR_ADDR --from $DELEGATOR_ADDR --network $NETWORK --gas auto --gas-adjustment 1.5 --fees 1000upokt -y

# Wait for confirmation
sleep 10

# Check updated rewards (should be 0 or very small)
UPDATED_REWARDS=$(pocketd query distribution rewards-by-validator $DELEGATOR_ADDR $VALIDATOR_ADDR --network $NETWORK --output json | jq -r '.rewards[0].amount // "0"')
echo "Rewards after withdrawal: $UPDATED_REWARDS uPOKT"
```

## Troubleshooting

### Common Issues

#### 1. No Validator Rewards Being Distributed

**Symptoms**: Distribution module balance not increasing, validator outstanding rewards stay at zero

**Possible Causes**:
- `mint_allocation_percentages.proposer` is set to 0
- No claim settlements occurring (no relay traffic)
- All validators have zero bonded stake

**Diagnostics**:
```bash
# Check tokenomics parameters
pocketd query tokenomics params --network <network>

# Check for recent settlements
pocketd query tx --events 'pocket.tokenomics.EventClaimSettled.num_relays>0' --network <network>

# Check validator stakes  
pocketd query staking validators --network <network>
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

**Symptoms**: Delegator reward balance not increasing despite validator receiving rewards

**Possible Causes**:
- High validator commission (100%)
- Recent delegation changes
- Rewards haven't accumulated enough to show

**Diagnostics**:
```bash
# Check validator commission rate
pocketd query staking validator <validator-addr> --network <network>

# Check delegation details
pocketd query staking delegation <delegator-addr> <validator-addr> --network <network>
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
# Distribution module
pocketd query bank balance cosmos1jv65s3grqf6v6jl3dp4t6c9t9rk99cd88lyufl upokt --network <network>

# Tokenomics module  
pocketd query bank balance cosmos1jv65s3grqf6v6jl3dp4t6c9t9rk99cd8w8zk8h upokt --network <network>
```

## Best Practices for Validators

### For Validator Operators

1. **Monitor Commission Settings**: Balance earning commission with attracting delegators
2. **Track Outstanding Rewards**: Regularly check accumulated rewards
3. **Watch Settlement Activity**: Monitor relay traffic and claim settlements in your network
4. **Set Up Alerting**: Monitor unusual reward distribution patterns

### For Delegators  

1. **Compare Commission Rates**: Choose validators with reasonable commission rates
2. **Monitor Reward Accumulation**: Track your rewards over time
3. **Regular Withdrawals**: Withdraw rewards periodically (they don't auto-compound)
4. **Diversify Delegations**: Consider spreading stake across multiple validators

### Sample Monitoring Script

```bash
#!/bin/bash

# Configuration
VALIDATOR_ADDR="<your-validator-address>"
DELEGATOR_ADDR="<your-delegator-address>"  
NETWORK="<your-network>"
ALERT_THRESHOLD=1000000  # Alert if rewards exceed 1M uPOKT

while true; do
    # Check validator outstanding rewards
    VAL_REWARDS=$(pocketd query distribution validator-outstanding-rewards $VALIDATOR_ADDR --network $NETWORK --output json | jq -r '.rewards[0].amount // "0"')
    
    # Check delegator rewards
    DEL_REWARDS=$(pocketd query distribution rewards-by-validator $DELEGATOR_ADDR $VALIDATOR_ADDR --network $NETWORK --output json | jq -r '.rewards[0].amount // "0"')
    
    # Check distribution module balance
    DIST_BALANCE=$(pocketd query bank balance cosmos1jv65s3grqf6v6jl3dp4t6c9t9rk99cd88lyufl upokt --network $NETWORK --output json | jq -r '.balance.amount')
    
    echo "$(date):"
    echo "  Validator Outstanding: $VAL_REWARDS uPOKT"
    echo "  Delegator Rewards: $DEL_REWARDS uPOKT"  
    echo "  Distribution Module: $DIST_BALANCE uPOKT"
    
    # Alert if rewards are high
    if [ "$DEL_REWARDS" -gt "$ALERT_THRESHOLD" ]; then
        echo "🚨 ALERT: Delegator rewards exceed threshold! Consider withdrawing."
    fi
    
    sleep 300  # Check every 5 minutes
done
```

This playbook provides the foundation for monitoring and understanding validator rewards in Pocket Network. For additional support, consult the network documentation or community channels.