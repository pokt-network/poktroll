---
sidebar_position: 6
title: "[WIP] E2E Relay Own Your Own Service"
---

:::danger This is a WIP by @olshansk

Intended as a sketchpad by @olshansk that will be productionized by @olshansk

:::

This guide walks you through setting up a RelayMiner for testing purposes using Anvil as the backend service.

- [Prerequisites](#prerequisites)
- [Account Preparation](#account-preparation)
  - [Create Local Accounts](#create-local-accounts)
  - [Import Accounts to Vultr Instance](#import-accounts-to-vultr-instance)
- [Service Setup](#service-setup)
  - [Create Service](#create-service)
  - [Create Application](#create-application)
  - [Create Supplier](#create-supplier)
- [Anvil Node Setup](#anvil-node-setup)
  - [Install Foundry](#install-foundry)
  - [Start Anvil](#start-anvil)
- [RelayMiner Configuration](#relayminer-configuration)
  - [Create RelayMiner Config](#create-relayminer-config)
  - [Start RelayMiner](#start-relayminer)
- [Testing](#testing)
  - [Send a Test Relay](#send-a-test-relay)
  - [Verify Claims](#verify-claims)
- [Next Steps](#next-steps)

## Prerequisites

Before starting, ensure you have:

1. **Vultr service setup**: Follow the [Vultr instance creation guide](https://dev.poktroll.com/operate/playbooks/vultr?_highlight=vult#create-the-vultr-instance)
2. **RC helpers configured**: Set up [RC helpers for Shannon environments](https://www.notion.so/buildwithgrove/Playbook-Streamlining-rc-helpers-for-Shannon-Alpha-Beta-Main-Network-Environments-152a36edfff680019314d468fad88864?source=copy_link)
3. **Funding account**: You need a `$FUNDING_ADDR` environment variable set with an account that has sufficient POKT to fund test accounts

## Account Preparation

### Create Local Accounts

Run the following commands on your local machine to create the required accounts:

```bash
# Create accounts
pocketd keys add olshansky_anvil_test_service_owner
pocketd keys add olshansky_anvil_test_app
pocketd keys add olshansky_anvil_test_gateway
pocketd keys add olshansky_anvil_test_supplier

# Export addresses
export OLSHANSKY_ANVIL_TEST_SERVICE_OWNER_ADDR=$(pocketd keys show olshansky_anvil_test_service_owner -a)
export OLSHANSKY_ANVIL_TEST_APP_ADDR=$(pocketd keys show olshansky_anvil_test_app -a)
export OLSHANSKY_ANVIL_TEST_GATEWAY_ADDR=$(pocketd keys show olshansky_anvil_test_gateway -a)
export OLSHANSKY_ANVIL_TEST_SUPPLIER_ADDR=$(pocketd keys show olshansky_anvil_test_supplier -a)

# Fund accounts
pocketd tx bank send $FUNDING_ADDR $OLSHANSKY_ANVIL_TEST_SERVICE_OWNER_ADDR 100000000upokt --network=beta --fees=100upokt --unordered --timeout-duration=5s --yes
pocketd tx bank send $FUNDING_ADDR $OLSHANSKY_ANVIL_TEST_APP_ADDR 100000000upokt --network=beta --fees=100upokt --unordered --timeout-duration=5s --yes
pocketd tx bank send $FUNDING_ADDR $OLSHANSKY_ANVIL_TEST_GATEWAY_ADDR 100000000upokt --network=beta --fees=100upokt --unordered --timeout-duration=5s --yes
pocketd tx bank send $FUNDING_ADDR $OLSHANSKY_ANVIL_TEST_SUPPLIER_ADDR 100000000upokt --network=beta --fees=100upokt --unordered --timeout-duration=5s --yes

# Export private keys
pocketd keys export olshansky_anvil_test_service_owner --unsafe --unarmored-hex --yes
pocketd keys export olshansky_anvil_test_app --unsafe --unarmored-hex --yes
pocketd keys export olshansky_anvil_test_gateway --unsafe --unarmored-hex --yes
pocketd keys export olshansky_anvil_test_supplier --unsafe --unarmored-hex --yes
```

### Import Accounts to Vultr Instance

SSH into your Vultr instance and import the accounts.

Replace `<hex>` with the actual hex private keys exported in the previous step:

```bash
ssh root@$VULTR_INSTANCE_IP

# Import accounts using the hex private keys from previous step
pocketd keys import-hex --keyring-backend=test olshansky_anvil_test_service_owner <hex>
pocketd keys import-hex --keyring-backend=test olshansky_anvil_test_app <hex>
pocketd keys import-hex --keyring-backend=test olshansky_anvil_test_gateway <hex>
pocketd keys import-hex --keyring-backend=test olshansky_anvil_test_supplier <hex>
```

## Service Setup

### Create Service

Create a new service on-chain for your Anvil test environment:

```bash
# Format: pocketd tx service add-service <service_id> <name> <compute_units_per_relay>
pocketd tx service add-service olshansky_anvil_test "Test service for olshansky by olshansky" 7 --keyring-backend=test --from=olshansky_anvil_test_service_owner --network=beta --yes --fees=200upokt
```

:::note

The value `7` represents compute units per relay for this service. Adjust based on your service's computational cost.

:::

### Create Application

Create the application configuration:

```bash
cat <<EOF > stake_app_config.yaml
stake_amount: 60000000000upokt  # 60,000 POKT minimum for testnet
service_ids:
  - "olshansky_anvil_test"
EOF
```

Stake the application:

```bash
pocketd tx application stake-application --config=stake_app_config.yaml --keyring-backend=test --from=olshansky_anvil_test_app --network=beta --yes --fees=200upokt --unordered --timeout-duration=1m
```

Verify the application:

```bash
pocketd query application show-application $(pocketd keys show olshansky_anvil_test_app -a --keyring-backend=test) --network=beta
```

### Create Supplier

Create the supplier configuration:

```bash
cat <<EOF > stake_supplier_config.yaml
owner_address: $(pocketd keys show olshansky_anvil_test_supplier -a --keyring-backend=test)
operator_address: $(pocketd keys show olshansky_anvil_test_supplier -a --keyring-backend=test)
stake_amount: 100000000upokt  # 100 POKT minimum for testnet
default_rev_share_percent:
  $(pocketd keys show olshansky_anvil_test_supplier -a --keyring-backend=test): 100
services:
  - service_id: "olshansky_anvil_test"
    endpoints:
      - publicly_exposed_url: http://$(curl ifconfig.me):8545  # Uses your public IP
        rpc_type: JSON_RPC
EOF
```

Stake the supplier:

```bash
pocketd tx supplier stake-supplier --config=stake_supplier_config.yaml --keyring-backend=test --from=olshansky_anvil_test_supplier --network=beta --yes --fees=200upokt --unordered --timeout-duration=1m
```

Verify the supplier:

```bash
pocketd query supplier show-supplier $(pocketd keys show olshansky_anvil_test_supplier -a --keyring-backend=test) --network=beta
```

## Anvil Node Setup

### Install Foundry

```bash
curl -L https://foundry.paradigm.xyz | bash
source ~/.foundry/bin
foundryup
```

### Start Anvil

Create a startup script:

```bash
cat <<EOF> start_anvil.sh
#!/bin/bash

# Run Anvil in background with nohup, redirecting output to anvil.log
nohup anvil --port 8545 > anvil.log 2>&1 &
echo "Anvil started on port 8545. Logs: anvil.log"
EOF

chmod +x start_anvil.sh
```

Start Anvil:

```bash
./start_anvil.sh
```

Verify Anvil is running:

```bash
# Check if Anvil process is running
ps aux | grep anvil

# View recent logs
tail -20 anvil.log
```

Test the connection:

```bash
curl -X POST http://127.0.0.1:8545 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc": "2.0", "id": 1, "method": "eth_blockNumber", "params": []}'
```

## RelayMiner Configuration

### Create RelayMiner Config

```bash
cat <<EOF> relay_miner_config.yaml
default_signing_key_names:
  - olshansky_anvil_test_supplier
smt_store_path: /root/.pocket/smt
pocket_node:
  query_node_rpc_url: https://sauron-rpc.beta.infra.pocket.network
  query_node_grpc_url: https://sauron-grpc.beta.infra.pocket.network:443
  tx_node_rpc_url: https://sauron-rpc.beta.infra.pocket.network
suppliers:
  - service_id: "olshansky_anvil_test" # change if not using Anvil
    service_config:
      backend_url: "http://127.0.0.1:8545" # change if not using Anvil
    listen_url: http://0.0.0.0:8545 # must match Supplier's public URL
metrics:
  enabled: false
  addr: :9090
pprof:
  enabled: false
  addr: :6060
EOF
```

### Start RelayMiner

Configure firewall to allow external connections on port 8545:

```bash
sudo ufw allow 8545/tcp
```

Start the RelayMiner:

```bash
pocketd relayminer start --config=relay_miner_config.yaml --chain-id=pocket-lego-testnet --keyring-backend=test --grpc-insecure=false
```

:::tip

Consider running the RelayMiner in a `tmux` or `screen` session, or as a systemd service for production use.

:::

## Testing

### Send a Test Relay

In a separate shell, send a test relay:

```bash
pocketd relayminer relay --keyring-backend=test  \
  --app=$(pocketd keys show olshansky_anvil_test_app -a --keyring-backend=test) \
  --supplier=$(pocketd keys show olshansky_anvil_test_supplier -a --keyring-backend=test) \
  --node=https://sauron-rpc.beta.infra.pocket.network \
  --grpc-addr=sauron-grpc.beta.infra.pocket.network:443 \
  --grpc-insecure=false \
  --payload="{\"jsonrpc\": \"2.0\", \"id\": 1, \"method\": \"eth_blockNumber\", \"params\": []}"
```

### Verify Claims

Check if your RelayMiner created any claims:

```bash
pocketd query txs --node=https://sauron-rpc.beta.infra.pocket.network \
        --query="tx.height>20000 AND message.action='/pocket.proof.MsgCreateClaim'" \
        --limit 10 --page 1 -o json | jq '[.txs[].tx.body.messages[] | select(."@type" == "/pocket.proof.MsgCreateClaim") | .supplier_operator_address] | unique'
```

## Next Steps

Your RelayMiner should now be running and processing relays.

**Monitoring and troubleshooting:**

- Monitor RelayMiner logs for incoming relay requests
- Check Anvil logs at `anvil.log` for backend activity
- Query claims periodically to verify relay processing
- Use `pocketd query proof list-claims --network=beta` to see all recent claims

**Common issues:**

- **Port conflicts**: Ensure port 8545 is not already in use (`netstat -tlnp | grep 8545`)
- **Firewall blocking**: Verify UFW allows port 8545 (`sudo ufw status`)
- **Session not started**: Relays only work during active sessions; check session timing with `pocketd query session get-session`
