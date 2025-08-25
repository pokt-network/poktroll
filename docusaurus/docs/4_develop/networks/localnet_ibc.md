---
sidebar_position: 3
title: LocalNet IBC Development
---

# LocalNet IBC Development & Debugging Guide

This guide provides copy-pasta commands for developing and debugging IBC functionality on LocalNet with multiple blockchain networks.

## Overview

LocalNet supports IBC connections between Pocket and other Cosmos-based chains including:
- **Agoric** (`agoriclocal`) - Smart contract platform
- **Axelar** (`axelar`) - Cross-chain infrastructure 
- **Osmosis** (`osmosis`) - DEX and AMM

All IBC operations use the Hermes relayer to facilitate cross-chain communication.

## Quick Setup

### Enable IBC in LocalNet

Edit `localnet_config.yaml`:

```yaml
ibc:
  enabled: true
  validator_configs:
    agoric:
      chain_id: agoriclocal
      chart_name: agoric-validator
      dockerfile_path: localnet/dockerfiles/agoric-validator.dockerfile
      enabled: true
      image_name: agoric
      port_forwards:
        - 46657:26657
        - 11090:9090
        - 40009:40009
      tilt_ui_name: Agoric Validator
      values_path: localnet/kubernetes/values-agoric.yaml
```

### Start LocalNet with IBC

```bash
# Start LocalNet with IBC enabled
make localnet_up

# Check IBC resources in Tilt UI
open http://localhost:10350
```

## IBC Access Points

### Validator Nodes

```bash
# Access Agoric shell interactively
make agoric_shell

# Access Axelar shell interactively  
make axelar_shell

# Access Osmosis shell interactively
make osmosis_shell
```

### Hermes Relayer

```bash
# Check relayer status
kubectl logs -f $(kubectl get pods | grep ibc-relayer | cut -f1 -d' ')

# View relayer configuration
kubectl exec -it $(kubectl get pods | grep ibc-relayer | cut -f1 -d' ') -- cat /root/.hermes/config.toml
```

## IBC Client, Connection & Channel Queries

### Pocket Network Queries

```bash
# List IBC clients on Pocket
make ibc_list_pocket_clients

# List IBC connections on Pocket
make ibc_list_pocket_connections

# List IBC channels on Pocket
make ibc_list_pocket_channels
```

### Agoric Network Queries

```bash
# List IBC clients on Agoric
make ibc_list_agoric_clients

# List IBC connections on Agoric
make ibc_list_agoric_connections

# List IBC channels on Agoric  
make ibc_list_agoric_channels
```

### Manual IBC Queries

```bash
# Query Pocket IBC state directly
pocketd q ibc client states --node=tcp://127.0.0.1:26657
pocketd q ibc connection connections --node=tcp://127.0.0.1:26657  
pocketd q ibc channel channels --node=tcp://127.0.0.1:26657

# Query IBC state on other networks using shell access
# For Agoric:
make agoric_shell
agd query ibc client states
agd query ibc connection connections
agd query ibc channel channels

# For Axelar:
make axelar_shell
axelard query ibc client states
axelard query ibc connection connections  
axelard query ibc channel channels

# For Osmosis:
make osmosis_shell
osmosisd query ibc client states
osmosisd query ibc connection connections
osmosisd query ibc channel channels
```

## Account Management

### Query Balances

```bash
# Query Pocket account balance
pocketd query bank balances $(pocketd keys show app1 -a) --node=tcp://127.0.0.1:26657

# Query Agoric account balance
make ibc_query_agoric_balance

# Query Axelar account balance
make ibc_query_axelar_balance

# Query Osmosis account balance
make ibc_query_osmosis_balance
```

### Fund Accounts

```bash
# Fund Agoric account with native tokens
make fund_agoric_account

# Fund Pocket account using faucet (if enabled)
curl -X POST http://localhost:8080/faucet \
  -H "Content-Type: application/json" \
  -d '{"address": "'"$(pocketd keys show app1 -a)"'", "coins": [{"denom": "upokt", "amount": "1000000"}]}'
```

## IBC Token Transfers

### Agoric ↔ Pocket Transfers

```bash
# Transfer from Agoric to Pocket
make ibc_test_transfer_agoric_to_pocket

# Transfer from Pocket to Agoric
make ibc_test_transfer_pocket_to_agoric
```

### Axelar ↔ Pocket Transfers

```bash
# Transfer from Axelar to Pocket
make ibc_test_transfer_axelar_to_pocket

# Transfer from Pocket to Axelar
make ibc_test_transfer_pocket_to_axelar
```

### Manual IBC Transfers

```bash
# Manual transfer from Pocket to Agoric
# First, get channel ID from: make ibc_list_pocket_channels
CHANNEL_ID="channel-0"  # Replace with actual channel ID
AGORIC_ACCOUNT="agoric1vaj34dfx94y6nvwt57dfyag5gfsp6eqjmvzu8c"

pocketd tx ibc-transfer transfer transfer \
  $CHANNEL_ID $AGORIC_ACCOUNT 1000upokt \
  --from=app1 \
  --keyring-backend=test \
  --network=local \
  --yes

# Manual transfer from any network to Pocket
# Get channel ID from: make ibc_list_<network>_channels
# Use network shell, then execute transfer command

# Example for Agoric to Pocket:
make agoric_shell
# Then: agd tx ibc-transfer transfer transfer \
#   channel-0 pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4 1000ubld \
#   --from=validator --keyring-backend=test --chain-id=agoriclocal --yes

# Example for Axelar to Pocket:
make axelar_shell  
# Then: axelard tx ibc-transfer transfer transfer \
#   channel-0 <pocket-address> 1000uaxl \
#   --from=validator --keyring-backend=test --chain-id=axelar --yes
```

## Transaction Debugging

### Query Transactions

```bash
# Query Pocket transaction
make query_tx TX_HASH=<hash>

# Query Agoric transaction
make agoric_query_tx TX_HASH=<hash>

# Query Axelar transaction
make axelar_query_tx TX_HASH=<hash>

# Query Osmosis transaction
make osmosis_query_tx TX_HASH=<hash>
```

### View Transaction Logs

```bash
# Pocket transaction logs
pocketd query tx <hash> --node=tcp://127.0.0.1:26657 --output=json | jq '.raw_log'

# Transaction logs on other networks using shell access
# Example for Agoric:
make agoric_shell
agd query tx <hash> --chain-id=agoriclocal --output=json | jq '.raw_log'

# Example for Axelar:
make axelar_shell
axelard query tx <hash> --chain-id=axelar --output=json | jq '.raw_log'
```

## IBC Packet Debugging

### Query Packet Commitments

```bash
# Query packet commitments on Pocket
pocketd q ibc channel packet-commitments transfer channel-0 --node=tcp://127.0.0.1:26657

# Query packet commitments on other networks using shell access
# For any network: make <network>_shell
# Then: <network-binary> query ibc channel packet-commitments transfer <channel-id>

# Example for Agoric:
make agoric_shell
agd query ibc channel packet-commitments transfer channel-0
```

### Query Packet Acknowledgements

```bash
# Query packet acknowledgements on Pocket
pocketd q ibc channel packet-acks transfer channel-0 --node=tcp://127.0.0.1:26657

# Query packet acknowledgements on other networks using shell access
# Example for Agoric:
make agoric_shell
agd query ibc channel packet-acks transfer channel-0
```

### Query Unreceived Packets

```bash
# Query unreceived packets on Pocket
pocketd q ibc channel unreceived-packets transfer channel-0 --node=tcp://127.0.0.1:26657

# Query unreceived packets on other networks using shell access
# Example for Agoric:
make agoric_shell
agd query ibc channel unreceived-packets transfer channel-0
```

## Hermes Relayer Operations

### Check Relayer Health

```bash
# View relayer logs
kubectl logs -f $(kubectl get pods | grep ibc-relayer | cut -f1 -d' ')

# Check relayer version
kubectl exec -it $(kubectl get pods | grep ibc-relayer | cut -f1 -d' ') -- hermes version

# Query chains configured in relayer
kubectl exec -it $(kubectl get pods | grep ibc-relayer | cut -f1 -d' ') -- hermes config validate
```

### Manual Relayer Commands

```bash
# Start relaying packets manually (if daemon not running)
kubectl exec -it $(kubectl get pods | grep ibc-relayer | cut -f1 -d' ') -- \
  hermes start

# Query pending packets
kubectl exec -it $(kubectl get pods | grep ibc-relayer | cut -f1 -d' ') -- \
  hermes query packet pending --chain pocket --port transfer --channel channel-0

# Clear packets manually
kubectl exec -it $(kubectl get pods | grep ibc-relayer | cut -f1 -d' ') -- \
  hermes clear packets --chain pocket --port transfer --channel channel-0
```

## IBC Setup and Connection Management

### Reset IBC Connections

```bash
# Restart IBC setup (reestablish connections)
make ibc_restart_setup

# Manual connection setup
# This triggers the IBC setup jobs in Tilt to reestablish connections
tilt trigger "🏗️ Pokt->Agoric"
tilt trigger "🏗️ Pokt->Axelar"  
tilt trigger "🏗️ Pokt->Osmosis"
```

### View IBC Configuration Files

```bash
# View Hermes relayer config
kubectl exec -it $(kubectl get pods | grep ibc-relayer | cut -f1 -d' ') -- \
  cat /root/.hermes/config.toml

# View IBC relayer Kubernetes config
cat localnet/kubernetes/values-ibc-relayer-config-pocket.yaml
cat localnet/kubernetes/values-ibc-relayer-config-agoriclocal.yaml

# View IBC validator configs
cat localnet/kubernetes/values-agoric.yaml
```

## Development Workflow

### Testing IBC Functionality

```bash
# 1. Start LocalNet with IBC
make localnet_up

# 2. Wait for all validators to be ready
kubectl get pods | grep -E "(validator|agoric|axelar|osmosis)"

# 3. Check IBC connections are established  
make ibc_list_pocket_channels
make ibc_list_agoric_channels

# 4. Test token transfer
make ibc_test_transfer_pocket_to_agoric

# 5. Query balances to verify transfer
make ibc_query_agoric_balance
pocketd query bank balances $(pocketd keys show app1 -a) --node=tcp://127.0.0.1:26657

# 6. Test reverse transfer
make ibc_test_transfer_agoric_to_pocket
```

### Hot Reloading with IBC

```bash
# IBC supports hot reloading - code changes will rebuild containers
# Watch for changes and rebuilds in Tilt UI: http://localhost:10350

# Manually restart IBC components if needed
kubectl rollout restart deployment/ibc-relayer
kubectl rollout restart deployment/agoric-validator

# Check logs after restart
kubectl logs -f deployment/ibc-relayer
```

## Configuration

### Enable/Disable IBC Networks

Edit `localnet_config.yaml`:

```yaml
ibc:
  enabled: true
  validator_configs:
    agoric:
      enabled: true      # Enable Agoric
      chain_id: agoriclocal
      # ... other config
    axelar:
      enabled: false     # Disable Axelar
      chain_id: axelar
      # ... other config
```

### IBC Relayer Configuration

Key configuration files:
- `localnet/kubernetes/values-ibc-relayer-common.yaml` - Common relayer settings
- `localnet/kubernetes/values-ibc-relayer-daemon.yaml` - Daemon mode settings
- `localnet/kubernetes/values-ibc-relayer-config-pocket.yaml` - Pocket chain config
- `localnet/kubernetes/values-ibc-relayer-config-agoriclocal.yaml` - Agoric chain config

## Monitoring and Observability

### IBC Metrics

```bash
# View IBC-related metrics in Grafana
open http://localhost:3003

# Check relayer metrics (if exposed)
kubectl port-forward deployment/ibc-relayer 3001:3001
curl http://localhost:3001/metrics
```

### Debug IBC Issues

```bash
# Check all IBC-related pods
kubectl get pods -l app.kubernetes.io/name=ibc-relayer
kubectl get pods | grep -E "(agoric|axelar|osmosis)-validator"

# Describe problematic resources
kubectl describe pod $(kubectl get pods | grep ibc-relayer | cut -f1 -d' ')

# Get events related to IBC
kubectl get events --field-selector reason=Failed | grep -i ibc

# Check resource usage
kubectl top pods | grep -E "(relayer|agoric|axelar|osmosis)"
```

## Troubleshooting

### Common IBC Issues

#### Connection Not Established

```bash
# Check if validators are running
kubectl get pods | grep validator

# Manually trigger IBC setup
make ibc_restart_setup

# Check relayer logs for errors
kubectl logs $(kubectl get pods | grep ibc-relayer | cut -f1 -d' ') | tail -100
```

#### Token Transfer Fails

```bash
# Check channel status
make ibc_list_pocket_channels | jq '.channels[] | select(.state == "STATE_OPEN")'

# Check if packets are stuck
pocketd q ibc channel packet-commitments transfer channel-0 --node=tcp://127.0.0.1:26657

# Clear pending packets
kubectl exec -it $(kubectl get pods | grep ibc-relayer | cut -f1 -d' ') -- \
  hermes clear packets --chain pocket --port transfer --channel channel-0
```

#### Relayer Not Working

```bash
# Restart relayer
kubectl rollout restart deployment/ibc-relayer

# Check relayer configuration
kubectl exec -it $(kubectl get pods | grep ibc-relayer | cut -f1 -d' ') -- \
  hermes health-check

# Validate config
kubectl exec -it $(kubectl get pods | grep ibc-relayer | cut -f1 -d' ') -- \
  hermes config validate
```

### Reset IBC State

```bash
# Complete IBC reset (destructive)
make localnet_down
kubectl delete pvc --all
make localnet_up

# Partial reset - just restart IBC components
kubectl delete pod $(kubectl get pods | grep ibc-relayer | cut -f1 -d' ')
kubectl delete pod $(kubectl get pods | grep agoric-validator | cut -f1 -d' ')
make ibc_restart_setup
```

## Useful Scripts

### Check IBC Channel Status

Create `/tmp/check_ibc_status.sh`:

```bash
#!/bin/bash
echo "=== Pocket IBC Channels ==="
make ibc_list_pocket_channels | jq '.channels[] | {channel_id: .channel_id, state: .state, counterparty: .counterparty}'

echo -e "\n=== Agoric IBC Channels ==="
make ibc_list_agoric_channels | jq '.channels[] | {channel_id: .channel_id, state: .state, counterparty: .counterparty}'

echo -e "\n=== IBC Relayer Status ==="
kubectl get pods | grep ibc-relayer
```

### Test IBC Transfer End-to-End

Create `/tmp/test_ibc_transfer.sh`:

```bash
#!/bin/bash
echo "Testing IBC transfer from Pocket to Agoric..."

# Get initial balances
echo "Initial Pocket balance:"
pocketd query bank balances $(pocketd keys show app1 -a) --node=tcp://127.0.0.1:26657

echo "Initial Agoric balance:"
make ibc_query_agoric_balance

# Perform transfer
echo "Transferring tokens..."
make ibc_test_transfer_pocket_to_agoric

# Wait for transfer to complete
sleep 10

# Check final balances
echo "Final Pocket balance:"
pocketd query bank balances $(pocketd keys show app1 -a) --node=tcp://127.0.0.1:26657

echo "Final Agoric balance:"
make ibc_query_agoric_balance
```

## Advanced IBC Development

### Adding New IBC Chains

To add support for a new IBC chain:

1. **Add validator configuration** in `localnet_config.yaml`:
```yaml
ibc:
  validator_configs:
    mychain:
      enabled: true
      chain_id: mychain-1
      chart_name: mychain-validator
      dockerfile_path: localnet/dockerfiles/mychain-validator.dockerfile
      image_name: mychain
      port_forwards:
        - "26677:26657"
        - "9095:9090"
      tilt_ui_name: MyChain Validator
      values_path: localnet/kubernetes/values-mychain.yaml
```

2. **Create Dockerfile** at `localnet/dockerfiles/mychain-validator.dockerfile`

3. **Create Helm values** at `localnet/kubernetes/values-mychain.yaml`

4. **Add relayer config** at `localnet/kubernetes/values-ibc-relayer-config-mychain-1.yaml`

5. **Add make targets** in `makefiles/ibc.mk`:
```makefile
.PHONY: mychain_shell
mychain_shell: check_kubectl check_docker_ps check_kind
	bash -c '\
		source ./tools/scripts/ibc-channels.sh && \
		kubectl_exec_grep_pod_interactive mychain-validator "bash" \
	'
```

This comprehensive guide should provide all the copy-pasta commands needed for IBC development and debugging on LocalNet!