# IBC Testing & Debugging Guide

This guide provides comprehensive strategies for testing IBC functionality and debugging common issues when working with Poktroll's cross-chain integrations.

## Testing Strategies

### 1. Local Development Testing

Use Poktroll's LocalNet for initial development and testing:

```bash
# Start LocalNet with IBC chains
make localnet_up

# Test basic connectivity
make ibc_test_transfer_pocket_to_axelar
make ibc_test_transfer_axelar_to_pocket
```

**Benefits**:
- ✅ Controlled environment
- ✅ Fast iteration cycles  
- ✅ Full visibility into all chains
- ✅ No external dependencies

### 2. Testnet Integration Testing

Test against live testnets for realistic conditions:

```bash
# Configure for testnet
export CHAIN_ID="pocket-testnet"
export RPC_ENDPOINT="https://testnet-rpc.poktroll.com"

# Test with real network conditions
hermes tx ft-transfer --dst-chain pocket-testnet --src-chain your-testnet \
  --src-port transfer --src-channel channel-0 \
  --amount 1000 --denom utoken
```

**Benefits**:
- ✅ Real network latency and conditions
- ✅ Production-like infrastructure
- ✅ Community relayer integration
- ✅ End-to-end validation

### 3. Mainnet Pre-deployment Testing

Final validation before mainnet launch:

```bash
# Minimal test amounts
# Coordinate with relayer operators
# Monitor all metrics closely
```

**Benefits**:
- ✅ Production environment validation
- ✅ Real economic incentives
- ✅ Full ecosystem integration

## Testing Scenarios

### Basic Functionality Tests

#### 1. Token Transfer Tests
```bash
# Test successful transfers
make ibc_test_transfer_pocket_to_<chain>
make ibc_test_transfer_<chain>_to_pocket

# Verify balances updated correctly
make acc_balance_query ACC=<recipient-address>
```

#### 2. Failed Transfer Tests  
```bash
# Test invalid recipient address
pocketd tx ibc-transfer transfer transfer channel-0 \
  invalid-address 1000upokt --from sender

# Test insufficient funds
pocketd tx ibc-transfer transfer transfer channel-0 \
  valid-address 999999999999upokt --from poor-sender
```

### Advanced Testing Scenarios

#### 1. High-Frequency Transfers
```bash
# Script for load testing
for i in {1..100}; do
  pocketd tx ibc-transfer transfer transfer channel-0 \
    recipient 100upokt --from sender &
done
wait
```

#### 2. Timeout Scenarios
```bash
# Configure short timeout for testing
pocketd tx ibc-transfer transfer transfer channel-0 \
  recipient 1000upokt --from sender \
  --packet-timeout-height 0-100  # Very short timeout
```

#### 3. Multi-hop Transfers (with PFM)
```bash
# Transfer through multiple chains
# Requires Packet Forward Middleware setup
pocketd tx ibc-transfer transfer transfer channel-0 \
  "chain-b|channel-1|final-recipient" 1000upokt --from sender
```

### Automated Testing

#### Integration Test Suite
Create automated tests for continuous validation:

```bash
#!/bin/bash
# test-ibc-integration.sh

set -e

echo "Starting IBC integration tests..."

# Test 1: Basic bi-directional transfers
echo "Testing basic transfers..."
make ibc_test_transfer_pocket_to_axelar
make ibc_test_transfer_axelar_to_pocket

# Test 2: Multiple counterparty chains
echo "Testing multiple chains..."
for chain in axelar osmosis agoric; do
  make ibc_test_transfer_pocket_to_${chain}
  make ibc_test_transfer_${chain}_to_pocket
done

# Test 3: Verify final balances
echo "Verifying balances..."
make ibc_query_all_balances

echo "All tests passed!"
```

## Debugging Guide

### Common Issues and Solutions

#### 1. Packets Not Relaying

**Symptoms**: Transaction succeeds on source chain but no tokens appear on destination

**Debugging Steps**:
```bash
# Check packet commitment on source chain
pocketd query ibc channel packet-commitments transfer channel-0

# Check packet receipt on destination chain  
<dest-chain>d query ibc channel packet-receipts transfer channel-0

# Check relayer logs
hermes query packet commitments --chain pocket --port transfer --channel channel-0
```

**Solutions**:
- Restart relayer service
- Check RPC endpoint connectivity
- Verify packet data availability
- Check for relayer account balance issues

#### 2. Timeout Errors

**Symptoms**: Packets timeout before being processed

**Debugging Steps**:
```bash
# Check timeout settings
pocketd query ibc-transfer params

# Check chain heights and block times
pocketd status | jq '.sync_info.latest_block_height'
<dest-chain>d status | jq '.sync_info.latest_block_height'
```

**Solutions**:
- Increase timeout height/timestamp
- Optimize relayer frequency
- Check for chain synchronization issues

#### 3. Channel State Issues

**Symptoms**: Channel appears closed or in unexpected state

**Debugging Steps**:
```bash
# Check channel state
pocketd query ibc channel channels

# Check connection state
pocketd query ibc connection connections

# Check client state
pocketd query ibc client states
```

**Solutions**:
- Update expired clients
- Re-establish channels if needed
- Check for misbehavior evidence

### Debugging Tools

#### 1. Hermes Diagnostic Commands
```bash
# Health check
hermes health-check

# Query channel details
hermes query channel end --chain pocket --port transfer --channel channel-0

# Query connection details  
hermes query connection end --chain pocket --connection connection-0

# Query client details
hermes query client state --chain pocket --client 07-tendermint-0
```

#### 2. Chain Query Commands
```bash
# Query IBC state
pocketd query ibc --help

# Query specific packet
pocketd query ibc channel packet-commitment transfer channel-0 1

# Query acknowledgment
pocketd query ibc channel packet-ack transfer channel-0 1
```

#### 3. Transaction Analysis
```bash
# Get detailed transaction info
pocketd query tx <hash> --output json | jq

# Check transaction events
pocketd query tx <hash> --output json | jq '.logs[].events[]'

# Filter IBC events
pocketd query tx <hash> --output json | jq '.logs[].events[] | select(.type | startswith("ibc"))'
```

### Performance Monitoring

#### Key Metrics to Track

1. **Transfer Success Rate**
   ```bash
   # Monitor successful vs failed transfers
   # Track timeout rates
   # Measure transfer completion times
   ```

2. **Relayer Performance**
   ```bash
   # Relayer uptime
   # Packet processing latency
   # Error rates by type
   ```

3. **Chain Health**
   ```bash
   # Block production rates
   # RPC response times
   # Node synchronization status
   ```

#### Monitoring Commands
```bash
# Watch for new IBC events
pocketd query txs --events 'send_packet.packet_src_channel=channel-0' --limit 10

# Monitor relayer balance
watch -n 30 "pocketd query bank balances <relayer-address>"

# Check chain sync status
watch -n 10 "pocketd status | jq '.sync_info'"
```

## Testing Best Practices

### 1. Test Environment Management
- Use dedicated test accounts for each test scenario
- Maintain separate test tokens for different test types
- Reset test environment regularly to ensure clean state

### 2. Progressive Testing
- Start with LocalNet for basic functionality
- Move to testnet for integration testing
- Perform limited mainnet testing before full deployment

### 3. Documentation
- Document all test scenarios and expected outcomes
- Maintain test case templates for regression testing
- Record debugging procedures for common issues

### 4. Automation
- Automate repetitive test scenarios
- Set up continuous integration for IBC functionality
- Implement monitoring alerts for production issues

## Emergency Procedures

### 1. Channel Emergency Shutdown
```bash
# If needed, close channel to stop packet flow
hermes tx chan-close-init --dst-chain pocket --src-chain counterparty \
  --dst-connection connection-0 --dst-port transfer
```

### 2. Relayer Emergency Stop
```bash
# Stop relayer service immediately
pkill hermes

# Or stop specific path
hermes clear packets --chain pocket --port transfer --channel channel-0
```

### 3. Recovery Procedures
```bash
# Clear stuck packets
hermes clear packets --chain pocket --port transfer --channel channel-0

# Re-establish connections if needed
hermes create connection --a-chain pocket --b-chain counterparty

# Resume normal operations
hermes start
```

---

**Need help?** Contact the Poktroll team via [Discord](https://discord.gg/pokt) or check the [troubleshooting section](./localnet.md#troubleshooting) for additional solutions.