# Chain Integration Guide

This guide walks you through connecting your blockchain to Poktroll via IBC, enabling seamless cross-chain interactions and access to Poktroll's decentralized API infrastructure.

## Prerequisites

Before starting the integration process, ensure your blockchain meets these requirements:

### Technical Requirements
- ✅ **IBC-enabled Cosmos SDK chain** (v0.47+ recommended)
- ✅ **Reliable RPC/REST infrastructure** with high availability
- ✅ **IBC transfer module** enabled via governance or genesis
- ✅ **Scalable infrastructure** capable of handling cross-chain traffic

### Infrastructure Requirements
- ✅ **Persistent RPC endpoints** (recommended: multiple endpoints for redundancy)
- ✅ **Archive node access** for historical packet data
- ✅ **Monitoring setup** for chain health and IBC metrics
- ✅ **Governance mechanism** for enabling IBC transfers (if not already enabled)

### Access Requirements
- ✅ **Relayer service** (your own or community relayer)
- ✅ **Test tokens** for initial transfer testing
- ✅ **Communication channels** with Poktroll team for coordination

## Integration Process

### Phase 1: Preparation

#### 1. Enable IBC Transfers
If not already enabled, IBC transfers must be activated through your chain's governance process:

```bash
# Check if IBC transfers are enabled
<your-chain>d query ibc-transfer params

# If disabled, submit governance proposal to enable
<your-chain>d tx gov submit-proposal param-change proposal.json
```

#### 2. Infrastructure Setup
Ensure your infrastructure meets production requirements:

- **RPC Endpoints**: Configure multiple endpoints with load balancing
- **Pruning Settings**: Maintain sufficient historical data for IBC operations
- **Monitoring**: Set up alerts for node health and IBC metrics

#### 3. Register Chain Metadata
Register your chain's bech32 prefix with [SLIP173](https://github.com/satoshilabs/slips/blob/master/slip-0173.md) if not already done.

### Phase 2: IBC Connection Setup

#### 1. Light Client Creation
Create IBC light clients between your chain and Poktroll:

```bash
# Create client on your chain tracking Poktroll
hermes create client --host-chain <your-chain-id> --reference-chain poktroll

# Create client on Poktroll tracking your chain  
hermes create client --host-chain poktroll --reference-chain <your-chain-id>
```

#### 2. Connection Establishment
Establish authenticated IBC connection:

```bash
# Create connection between chains
hermes create connection --a-chain <your-chain-id> --b-chain poktroll
```

#### 3. Channel Creation
Open transfer channel for token transfers:

```bash
# Create transfer channel
hermes create channel --a-chain <your-chain-id> --a-connection connection-0 \
  --a-port transfer --b-port transfer
```

### Phase 3: Relayer Configuration

#### Option A: Community Relayers
For production deployments, consider using established relayer services:

1. **Contact Community Relayers**
   - Review [available relayer services](https://docs.osmosis.zone/osmosis-core/relaying/ibc-relayers-list/)
   - Contact relayers directly for custom configurations
   - Provide your chain details and requirements

2. **Coordinate Setup**
   - Share RPC endpoints and chain information
   - Coordinate channel creation timing
   - Test connectivity before going live

#### Option B: Self-Hosted Relayer
Set up your own Hermes relayer:

1. **Install Hermes**
   ```bash
   cargo install ibc-relayer-cli --bin hermes
   ```

2. **Configure Hermes**
   ```toml
   # ~/.hermes/config.toml
   [global]
   log_level = 'info'

   [[chains]]
   id = '<your-chain-id>'
   rpc_addr = 'https://your-rpc-endpoint.com'
   grpc_addr = 'https://your-grpc-endpoint.com'
   # ... additional configuration

   [[chains]]  
   id = 'poktroll'
   rpc_addr = 'https://poktroll-rpc-endpoint.com'
   grpc_addr = 'https://poktroll-grpc-endpoint.com'
   # ... additional configuration
   ```

3. **Start Relaying**
   ```bash
   hermes start
   ```

### Phase 4: Testing & Validation

#### 1. Test Token Transfers
Perform comprehensive bi-directional testing:

```bash
# Test transfer from your chain to Poktroll
<your-chain>d tx ibc-transfer transfer transfer channel-0 \
  pokt1receiver... 1000utoken --from sender

# Test transfer from Poktroll to your chain  
pocketd tx ibc-transfer transfer transfer channel-0 \
  <your-prefix>1receiver... 1000upokt --from sender
```

#### 2. Verify Packet Processing
Monitor relayer logs and confirm successful packet processing:

```bash
# Check Hermes logs for successful relay
hermes health-check
hermes query packet commitments --chain <your-chain-id> --port transfer --channel channel-0
```

#### 3. Integration Testing
Test various scenarios:
- ✅ Small amount transfers
- ✅ Large amount transfers  
- ✅ Failed transfer scenarios (invalid addresses, insufficient funds)
- ✅ Timeout scenarios
- ✅ High-frequency transfers

### Phase 5: Production Deployment

#### 1. Monitoring Setup
Implement comprehensive monitoring:

- **Chain Health**: Block production, validator status
- **IBC Metrics**: Packet success rates, timeout rates
- **Relayer Health**: Relayer uptime, error rates
- **Balance Monitoring**: Relayer account balances

#### 2. Documentation
Create integration documentation including:
- Connection details (chain ID, channels)
- Supported token denominations
- Transfer limits and fees
- Emergency procedures

#### 3. Community Announcement
Coordinate with both communities to announce the integration:
- Technical details and capabilities
- Usage instructions for end users
- Support channels for issues

## Integration Checklist

Use this checklist to track your integration progress:

### Pre-Integration
- [ ] IBC transfers enabled on your chain
- [ ] Infrastructure meets requirements
- [ ] Relayer solution identified
- [ ] Test environment prepared

### Technical Setup  
- [ ] Light clients created successfully
- [ ] IBC connection established
- [ ] Transfer channel opened
- [ ] Relayer configured and running

### Testing
- [ ] Bi-directional transfers tested
- [ ] Packet processing verified
- [ ] Edge cases tested
- [ ] Performance validated

### Production
- [ ] Monitoring implemented
- [ ] Documentation completed
- [ ] Community notified
- [ ] Support channels established

## Support & Resources

### Technical Support
- **Poktroll Team**: Contact via [Discord](https://discord.gg/pokt) or [Telegram](https://t.me/POKTnetwork)
- **Documentation**: [Poktroll Developer Docs](/)
- **GitHub**: [Report issues](https://github.com/pokt-network/poktroll/issues)

### IBC Resources
- [Cosmos IBC Documentation](https://ibc.cosmos.network/)
- [Hermes Relayer Guide](https://hermes.informal.systems/)
- [IBC Protocol Specification](https://github.com/cosmos/ibc)

### Community
- **Cosmos IBC Gang**: Join the IBC community for support and updates
- **Relayer Directory**: Find community relayers and connect with operators

---

**Ready to integrate?** Start with the [LocalNet development environment](./localnet.md) to test your integration before moving to production.