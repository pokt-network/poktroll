# Interchain Accounts (ICA)

Interchain Accounts enable a blockchain to control an account on a remote chain, allowing for cross-chain governance, staking, and complex multi-chain operations. This powerful IBC feature extends blockchain functionality beyond simple token transfers.

## What are Interchain Accounts?

ICA allows one blockchain (the "controller") to own and operate an account on another blockchain (the "host") through IBC. The controller chain can execute arbitrary transactions on the host chain as if it were a native user.

### Key Concepts

- **Controller Chain**: The chain that owns and controls the interchain account
- **Host Chain**: The chain where the interchain account exists and executes transactions  
- **Interchain Account**: A special account on the host chain controlled by the controller chain
- **Authentication Module**: Module on controller chain that manages ICA operations

### Use Cases

- **Cross-chain Governance**: Vote on governance proposals across multiple chains
- **Multi-chain Staking**: Stake tokens on multiple chains from a single interface
- **Automated Operations**: Execute complex multi-chain workflows programmatically
- **Cross-chain DeFi**: Participate in DeFi protocols across different chains
- **Portfolio Management**: Manage assets across multiple chains from one location

## ICA Architecture

```
Controller Chain                    Host Chain
┌─────────────────┐                ┌──────────────────┐
│  Auth Module    │                │                  │
│  ┌───────────┐  │                │                  │
│  │Application│  │   IBC Packet   │  ┌─────────────┐ │
│  │           │◄─┼────────────────┼─►│Interchain   │ │
│  │  Logic    │  │                │  │Account      │ │
│  └───────────┘  │                │  └─────────────┘ │
│                 │                │                  │
│ ICA Controller  │                │   ICA Host       │
│ Module          │                │   Module         │
└─────────────────┘                └──────────────────┘
```

### Components

1. **ICA Controller Module**: Manages outgoing ICA transactions from controller chain
2. **ICA Host Module**: Processes incoming ICA transactions on host chain  
3. **Authentication Module**: Custom module implementing ICA logic on controller chain
4. **Interchain Account**: The actual account on the host chain

## ICA in Poktroll

### Current Status

Poktroll supports ICA functionality with the following configuration:

```bash
# Check ICA parameters
pocketd query interchain-accounts controller params
pocketd query interchain-accounts host params

# List ICA connections
pocketd query interchain-accounts controller interchain-accounts <connection-id>
```

### Supported Operations

Poktroll's ICA implementation supports:

- ✅ **Account Registration**: Create interchain accounts on remote chains
- ✅ **Transaction Execution**: Execute arbitrary transactions via ICA  
- ✅ **Multi-chain Governance**: Participate in governance across chains
- ✅ **Staking Operations**: Delegate/undelegate tokens on remote chains
- 🚧 **Custom Modules**: Application-specific ICA functionality (in development)

## Setting Up ICA

### Prerequisites

Both controller and host chains must:
- Support IBC
- Have ICA modules enabled
- Have an established IBC connection
- Support the specific transaction types you want to execute

### 1. Register Interchain Account

From the controller chain, register an account on the host chain:

```bash
# Register ICA on host chain
pocketd tx interchain-accounts controller register <connection-id> \
  --from <controller-account> \
  --gas auto
```

### 2. Query Interchain Account Address

```bash
# Get the ICA address on host chain
pocketd query interchain-accounts controller interchain-account \
  <controller-address> <connection-id>
```

### 3. Fund Interchain Account

The ICA needs tokens on the host chain to pay for transaction fees:

```bash
# Send tokens to ICA via IBC transfer
pocketd tx ibc-transfer transfer transfer <channel-id> \
  <ica-address> 1000000utoken \
  --from <sender>
```

## Using ICA

### Basic Transaction Execution

Execute transactions on the host chain via ICA:

```bash
# Example: Delegate tokens on host chain
pocketd tx interchain-accounts controller send-tx <connection-id> \
  '{
    "body": {
      "messages": [{
        "@type": "/cosmos.staking.v1beta1.MsgDelegate",
        "delegator_address": "<ica-address>",
        "validator_address": "<validator-address>", 
        "amount": {"denom": "utoken", "amount": "1000000"}
      }]
    }
  }' \
  --from <controller-account>
```

### Governance Participation

Vote on governance proposals across chains:

```bash
# Vote on proposal via ICA
pocketd tx interchain-accounts controller send-tx <connection-id> \
  '{
    "body": {
      "messages": [{
        "@type": "/cosmos.gov.v1beta1.MsgVote",
        "proposal_id": "1",
        "voter": "<ica-address>",
        "option": "VOTE_OPTION_YES"
      }]
    }
  }' \
  --from <controller-account>
```

### Batch Operations

Execute multiple transactions in a single ICA packet:

```bash
# Batch delegate to multiple validators
pocketd tx interchain-accounts controller send-tx <connection-id> \
  '{
    "body": {
      "messages": [
        {
          "@type": "/cosmos.staking.v1beta1.MsgDelegate",
          "delegator_address": "<ica-address>",
          "validator_address": "<validator1>",
          "amount": {"denom": "utoken", "amount": "500000"}
        },
        {
          "@type": "/cosmos.staking.v1beta1.MsgDelegate", 
          "delegator_address": "<ica-address>",
          "validator_address": "<validator2>",
          "amount": {"denom": "utoken", "amount": "500000"}
        }
      ]
    }
  }' \
  --from <controller-account>
```

## LocalNet ICA Testing

```
// TODO(@bryanchriswhite)
```

## Advanced ICA Features

```
// TODO(@bryanchriswhite)
```

## Troubleshooting ICA

### Common Issues

#### 1. ICA Registration Fails

**Symptoms**: Account registration returns error

**Debug Steps**:
```bash
# Check connection status
pocketd query ibc connection connections

# Verify ICA parameters
pocketd query interchain-accounts controller params
<host-chain>d query interchain-accounts host params
```

**Solutions**:
- Ensure IBC connection is established and open
- Verify both chains support ICA
- Check that ICA is enabled in chain parameters

#### 2. Transaction Execution Fails

**Symptoms**: ICA transactions fail on host chain

**Debug Steps**:
```bash
# Check ICA account balance
<host-chain>d query bank balances <ica-address>

# Verify transaction format
echo '<transaction-json>' | jq

# Check host chain logs for errors
```

**Solutions**:
- Ensure ICA has sufficient balance for fees
- Verify transaction message format is correct
- Check that message types are supported on host chain

#### 3. Authentication Issues

**Symptoms**: ICA operations rejected due to authentication

**Debug Steps**:
```bash
# Verify ICA ownership
pocketd query interchain-accounts controller interchain-account \
  <owner-address> <connection-id>

# Check channel state
pocketd query ibc channel channels
```

**Solutions**:
- Ensure correct owner address is used
- Verify ICA channel is open and active
- Check authentication module configuration

### Monitoring ICA Operations

```bash
# Monitor ICA transactions
pocketd query txs --events 'submit_msgs.packet_src_channel=channel-0'

# Check ICA account activity
<host-chain>d query txs --events 'message.sender=<ica-address>'

# Monitor ICA channel health
pocketd query ibc channel channels | grep ica
```

## Best Practices

### 1. Security Considerations
- **Principle of Least Privilege**: Only grant necessary permissions to ICA
- **Key Management**: Secure controller account private keys
- **Transaction Validation**: Validate all ICA transactions before execution
- **Regular Audits**: Monitor ICA activity for unexpected behavior

### 2. Error Handling
- **Timeout Management**: Set appropriate timeouts for ICA operations
- **Retry Logic**: Implement retry mechanisms for failed transactions
- **Fallback Procedures**: Have contingency plans for ICA failures

### 3. Performance Optimization
- **Batch Operations**: Group multiple transactions when possible
- **Fee Management**: Monitor and optimize transaction fees
- **Connection Health**: Maintain healthy IBC connections for ICA

### 4. Monitoring and Alerting
- **Balance Monitoring**: Track ICA account balances
- **Operation Success**: Monitor ICA transaction success rates
- **Channel Health**: Alert on ICA channel issues

---

**Next Steps**: Explore [testing strategies](./testing.md) for comprehensive ICA validation or return to the [IBC overview](./index.md) for other cross-chain features.