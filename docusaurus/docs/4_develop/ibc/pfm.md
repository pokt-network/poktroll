# Packet Forward Middleware (PFM)

Packet Forward Middleware enables multi-hop IBC transfers, allowing tokens to be transferred across multiple chains in a single transaction. This powerful feature simplifies cross-chain workflows and enables complex routing scenarios.

## What is PFM?

PFM is an IBC middleware that intercepts IBC transfer packets and forwards them to additional destinations based on metadata in the packet memo field. Instead of requiring multiple separate transactions, users can specify a complete multi-hop route in a single transfer.

### Benefits
- **Single Transaction**: Multi-hop transfers in one transaction
- **Reduced Complexity**: No need to monitor intermediate chains
- **Atomic Operations**: All hops succeed or fail together
- **Lower Fees**: Reduced transaction fees compared to multiple transfers
- **Better UX**: Simplified user experience for complex routes
- **Enhanced Resilience**: Multipath routing prevents tokens from getting stuck when IBC clients fail or become tombstoned, as alternative routes through different chains remain available

## How PFM Works

```
User Chain A ─────► Chain B (with PFM) ─────> Chain C ─────► Final Destination
     │                    │                       │                │
     └── Transfer ───────►└── Forward ───────────►└── Forward ────►└── Receive
```

1. **Initial Transfer**: User initiates transfer from Chain A to Chain B
2. **PFM Processing**: Chain B's PFM middleware processes the packet
3. **Forward**: Chain B forwards the packet to Chain C based on memo instructions
4. **Final Delivery**: Tokens reach the final destination

## PFM Configuration in Poktroll

### Supported Routes

Poktroll supports PFM on the following routes:

```bash
# Direct PFM routes through Poktroll
make ibc_pfm_test_osmosis_to_axelar  # Osmosis → Poktroll → Axelar
make ibc_pfm_test_axelar_to_osmosis  # Axelar → Poktroll → Osmosis

# Additional routes (check current LocalNet config)
make ibc_pfm_test_agoric_to_osmosis  # Agoric → Poktroll → Osmosis
```

### LocalNet Testing

Test PFM functionality in your local development environment:

```bash
# Start LocalNet with PFM enabled
make localnet_up

# Verify PFM is configured
make ibc_list_pocket_channels | grep transfer

# Test multi-hop transfer
make ibc_pfm_test_osmosis_to_axelar
```

## Using PFM

### Transfer Syntax

PFM uses the memo field to specify forwarding instructions:

```json
{
  "forward": {
    "receiver": "final-recipient-address",
    "port": "transfer",
    "channel": "channel-X",
    "timeout": "10m",
    "retries": 3,
    "next": {
      // Optional: additional hops
    }
  }
}
```

### Basic Multi-hop Transfer

Transfer tokens from Chain A through Poktroll to Chain C:

```bash
# Example: Osmosis → Poktroll → Axelar
osmosisd tx ibc-transfer transfer transfer channel-0 \
  pokt1intermediate... 1000uosmo \
  --memo '{"forward":{"receiver":"axelar1final...","port":"transfer","channel":"channel-1"}}' \
  --from sender
```

### Complex Multi-hop with Multiple Forwards

```bash
# Chain A → Chain B → Chain C → Chain D
chaind tx ibc-transfer transfer transfer channel-0 \
  chain-b-intermediate 1000utoken \
  --memo '{"forward":{"receiver":"chain-c-intermediate","port":"transfer","channel":"channel-1","next":{"forward":{"receiver":"chain-d-final","port":"transfer","channel":"channel-2"}}}}' \
  --from sender
```

## PFM Make Targets

Poktroll provides convenient make targets for testing PFM functionality:

### Available PFM Tests

```bash
# Test PFM routes through Poktroll
make ibc_pfm_test_osmosis_to_axelar
make ibc_pfm_test_axelar_to_osmosis
make ibc_pfm_test_agoric_to_osmosis

# Query PFM state and routing
make ibc_pfm_query_routing
make ibc_pfm_query_statistics
```

### Custom PFM Testing

```bash
# Test custom PFM route
make ibc_pfm_test_custom SOURCE=<source-chain> DEST=<dest-chain> \
  AMOUNT=<amount> DENOM=<denom>
```

## Configuration

### Enabling PFM

PFM must be enabled in the IBC application stack:

```go
// In app.go
app.IBCKeeper = ibckeeper.NewKeeper(
    appCodec, keys[ibchost.StoreKey], app.GetSubspace(ibchost.ModuleName),
    app.StakingKeeper, app.UpgradeKeeper, scopedIBCKeeper,
)

// Add PFM middleware
app.TransferKeeper = ibctransferkeeper.NewKeeper(
    appCodec, keys[ibctransfertypes.StoreKey], app.GetSubspace(ibctransfertypes.ModuleName),
    app.IBCKeeper.ChannelKeeper, app.IBCKeeper.ChannelKeeper, &app.IBCKeeper.PortKeeper,
    app.AccountKeeper, app.BankKeeper, scopedTransferKeeper,
)

// Wrap with PFM middleware
app.PFMModule = pfm.NewAppModule(app.TransferKeeper)
```

### Channel Configuration

Ensure channels support PFM by using appropriate channel ordering:

```bash
# Create unordered channel for PFM
hermes create channel --a-chain source --b-chain destination \
  --a-port transfer --b-port transfer --channel-version ics20-1 \
  --order unordered
```

## Troubleshooting PFM

### Common Issues

#### 1. PFM Not Processing Forwards

**Symptoms**: Tokens stop at intermediate chain instead of continuing

**Debug Steps**:
```bash
# Check if PFM is enabled on intermediate chain
pocketd query ibc-transfer params

# Verify memo format
echo '{"forward":{"receiver":"dest","port":"transfer","channel":"channel-1"}}' | jq
```

**Solutions**:
- Verify PFM middleware is properly configured
- Check memo field syntax and encoding
- Ensure destination channel exists and is open

#### 2. Timeout Issues in Multi-hop

**Symptoms**: Transfers timeout during forwarding process

**Debug Steps**:
```bash
# Check timeout settings
pocketd query tx <hash> --output json | jq '.logs[].events[] | select(.type=="timeout_packet")'

# Monitor chain heights
for chain in source intermediate dest; do
  echo "$chain: $(${chain}d status | jq '.sync_info.latest_block_height')"
done
```

**Solutions**:
- Increase timeout values for multi-hop transfers
- Ensure all chains in the route are producing blocks
- Check relayer performance on all hops

#### 3. Invalid Routing Configuration

**Symptoms**: PFM rejects forwards with routing errors

**Debug Steps**:
```bash
# Verify channel mapping
pocketd query ibc channel channels | jq '.channels[] | select(.port_id=="transfer")'

# Check PFM routing table
pocketd query pfm routes
```

**Solutions**:
- Verify channel IDs are correct for each hop
- Check port IDs match expected values
- Ensure channels are in OPEN state

### Debugging Commands

```bash
# Query PFM statistics
pocketd query pfm stats

# Check forwarding history
pocketd query pfm forwards --limit 10

# Monitor PFM events
pocketd query txs --events 'forward_packet.src_channel=channel-0'
```

## Protecting Against Client Failures

PFM provides critical protection against **token stranding** when IBC clients fail or become tombstoned.

### The Problem: Dead Clients

Consider Chain A and Chain B with a direct IBC connection:

```bash
# Normal operation - direct transfers work
Chain A ◄─────────────────────────────────────────────────────────► Chain B
         (Client working - transfers flow freely)

# Client failure - tokens get stranded
Chain A ◄──────────────────── X ──────────────────────────────────► Chain B
         (Client tombstoned - tokens stuck on intermediate chain!)
```

When the Chain A↔Chain B client fails, any tokens that were mid-transfer or on the "wrong" chain cannot return home if/until the client is restored.

### The Solution: PFM Routing

PFM enables an alternative path through chains with working clients:

```bash
# Route through intermediate chain when direct client fails
Chain A ◄─────────────────────────────────────────────────────────► Chain B
         X Direct client failed

Chain A ◄─────────► Chain C ◄─────────► Chain B
         Working     Working     Working
```

### Practical Recovery

```bash
# Tokens stranded on Chain A due to failed Chain A→Chain B client
# Use PFM to route through Chain C instead:
chaind tx ibc-transfer transfer transfer channel-c \
  chainc1intermediate... 1000utoken \
  --memo '{"forward":{"receiver":"chainb1final...","port":"transfer","channel":"channel-b"}}' \
  --from sender

# Result: Chain A → Chain C → Chain B (bypassing the failed direct client)
```

This resilience makes PFM essential for production environments where token accessibility must be maintained even during client failures.

## Best Practices

### 1. Route Planning
- **Minimize Hops**: Use the shortest path possible
- **Verify Connectivity**: Ensure all channels in route are stable
- **Test Thoroughly**: Test complete routes before production use
- **Plan Alternatives**: Identify backup routes for critical transfers

### 2. Timeout Management
- **Conservative Timeouts**: Use generous timeout values for multi-hop
- **Chain Awareness**: Account for different block times across chains
- **Monitoring**: Monitor timeout rates and adjust accordingly

### 3. Error Handling
- **Graceful Failures**: Implement proper error handling for failed forwards
- **Retry Logic**: Consider retry mechanisms for transient failures
- **Monitoring**: Track forward success rates and error patterns

### 4. Security Considerations
- **Validate Recipients**: Verify recipient addresses for each hop
- **Limit Exposure**: Don't forward to untrusted chains
- **Monitor Flows**: Track large or unusual forwarding patterns

## Advanced PFM Features

### Custom Forwarding Logic

```json
{
  "forward": {
    "receiver": "intermediate-address",
    "port": "transfer", 
    "channel": "channel-1",
    "timeout": "10m",
    "retries": 3,
    "next": {
      "forward": {
        "receiver": "final-address",
        "port": "transfer",
        "channel": "channel-2",
        "timeout": "5m"
      }
    }
  }
}
```

### Conditional Forwarding

```json
{
  "forward": {
    "receiver": "conditional-address",
    "port": "transfer",
    "channel": "channel-1", 
    "condition": {
      "min_amount": "1000",
      "max_amount": "10000"
    }
  }
}
```

## Integration Examples

### Frontend Integration

```javascript
// Example: Osmosis → Poktroll → Axelar transfer
const memo = {
  forward: {
    receiver: "axelar1final...",
    port: "transfer", 
    channel: "channel-1",
    timeout: "10m"
  }
};

const msg = {
  typeUrl: "/ibc.applications.transfer.v1.MsgTransfer",
  value: {
    sourcePort: "transfer",
    sourceChannel: "channel-0", 
    token: { denom: "uosmo", amount: "1000000" },
    sender: "osmo1sender...",
    receiver: "pokt1intermediate...",
    memo: JSON.stringify(memo)
  }
};
```

### CLI Integration

```bash
#!/bin/bash
# Multi-hop transfer script

SOURCE_CHAIN="osmosis"
DEST_CHAIN="axelar"
AMOUNT="1000000"
DENOM="uosmo"

# Construct memo for PFM
MEMO='{"forward":{"receiver":"'$FINAL_RECIPIENT'","port":"transfer","channel":"'$DEST_CHANNEL'"}}'

# Execute transfer
${SOURCE_CHAIN}d tx ibc-transfer transfer transfer $SOURCE_CHANNEL \
  $INTERMEDIATE_RECIPIENT $AMOUNT$DENOM \
  --memo "$MEMO" \
  --from $SENDER
```

---

**Next Steps**: Explore [Interchain Accounts (ICA)](./ica.md) for cross-chain account control or return to the [IBC overview](./index.md) for other integration options.