---
sidebar_position: 2
title: Docker Compose Cheat Sheet
---

import ReactPlayer from "react-player";

# Docker Compose Cheat Sheet <!-- omit in toc --> <!-- omit in toc -->

- [Results](#results)
- [Deploy your server](#deploy-your-server)
- [Install Dependencies](#install-dependencies)
- [Retrieve the source code](#retrieve-the-source-code)
- [Update your environment](#update-your-environment)
- [Start up the full node](#start-up-the-full-node)
- [Create new addresses for all your accounts and update .env](#create-new-addresses-for-all-your-accounts-and-update-env)
- [Fund your accounts](#fund-your-accounts)
- [Stake a Supplier \& Deploy a RelayMiner](#stake-a-supplier--deploy-a-relayminer)
- [Stake an Application \& Deploy an AppGate Server](#stake-an-application--deploy-an-appgate-server)
- [Send a Relay](#send-a-relay)
  - [Ensure you get a response](#ensure-you-get-a-response)
- [\[BONUS\] Deploy a PATH Gateway](#bonus-deploy-a-path-gateway)
- [Managing a re-genesis](#managing-a-re-genesis)
  - [Full Nodes](#full-nodes)
  - [Fund the same accounts](#fund-the-same-accounts)
    - [Faucet is not ready and you need to fund the accounts manually](#faucet-is-not-ready-and-you-need-to-fund-the-accounts-manually)
  - [Start the RelayMiner](#start-the-relayminer)
  - [Start the AppGate Server](#start-the-appgate-server)
  - [Re-stake the gateway](#re-stake-the-gateway)

## Results

This is a text heavy walkthrough, but if all goes well, you should have something like the following:

<ReactPlayer
  playing={false}
  controls
  url="https://github.com/user-attachments/assets/11f5ae68-f8c1-4e12-99ec-641495c2dfb7"
/>

## Deploy your server

1. Go to [vultr's console](https://my.vultr.com/deploy)
2. Choose `Cloud Compute - Shared CPU`
3. Choose `Debian 12 x64`
4. Select `AMD High Performance`
5. Choose the `100GB NVMe` storage w/ `4GB` memory and `2 vCPU`
6. Disable `Auto Backups`
7. Deploy

## Install Dependencies

See [docker's official instructions here](https://docs.docker.com/engine/install/debian/).

Prepare the system:

```bash
# Add Docker's official GPG key:
sudo apt-get update
sudo apt-get install ca-certificates curl
sudo install -m 0755 -d /etc/apt/keyrings
sudo curl -fsSL https://download.docker.com/linux/debian/gpg -o /etc/apt/keyrings/docker.asc
sudo chmod a+r /etc/apt/keyrings/docker.asc

# Add the repository to Apt sources:
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian \
  $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt-get update

# Check if UFW is installed and add rules if it is
if command -v ufw > /dev/null 2>&1; then
    sudo ufw allow from 172.16.0.0/12
    sudo ufw allow from 192.168.0.0/16
    echo "UFW rules added for Docker networks"
else
    echo "UFW is not installed, skipping firewall configuration"
fi
```

And then install docker:

```bash
sudo apt-get install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
```

## Retrieve the source code

Then pull the github repo

```bash
mkdir ~/workspace && cd ~/workspace
git clone https://github.com/pokt-network/poktroll-docker-compose-example.git
cd poktroll-docker-compose-example
```

## Update your environment

```bash
cp .env.sample .env

EXTERNAL_IP=$(curl -4 ifconfig.me/ip)
sed -i -e s/NODE_HOSTNAME=/NODE_HOSTNAME=$EXTERNAL_IP/g .env

echo "source $(pwd)/helpers.sh" >> ~/.bashrc
echo "source $(pwd)/.env" >> ~/.bashrc
source ~/.bashrc
```

## Start up the full node

```bash
docker compose up -d full-node
# Optional: watch the block height sync up & logs
docker logs -f --tail 100 full-node
watch_height
```

## Create new addresses for all your accounts and update .env

Supplier:

```bash
poktrolld keys add supplier > /tmp/supplier

mnemonic=$(tail -n 1 /tmp/supplier | tr -d '\r'); sed -i "s|SUPPLIER_MNEMONIC=\".*\"|SUPPLIER_MNEMONIC=\"$mnemonic\"|" .env

address=$(awk '/address:/{print $3; exit}' /tmp/supplier | tr -d '\r'); sed -i "s|SUPPLIER_ADDR=\".*\"|SUPPLIER_ADDR=\"$address\"|g" .env
```

Application:

```bash
poktrolld keys add application > /tmp/application

mnemonic=$(tail -n 1 /tmp/application | tr -d '\r'); sed -i "s|APPLICATION_MNEMONIC=\".*\"|APPLICATION_MNEMONIC=\"$mnemonic\"|" .env

address=$(awk '/address:/{print $3; exit}' /tmp/application | tr -d '\r'); sed -i "s|APPLICATION_ADDR=\".*\"|APPLICATION_ADDR=\"$address\"|g" .env
```

Gateway:

```bash
poktrolld keys add gateway > /tmp/gateway

mnemonic=$(tail -n 1 /tmp/gateway | tr -d '\r'); sed -i "s|GATEWAY_MNEMONIC=\".*\"|GATEWAY_MNEMONIC=\"$mnemonic\"|" .env

address=$(awk '/address:/{print $3; exit}' /tmp/gateway | tr -d '\r'); sed -i "s|GATEWAY_ADDR=\".*\"|GATEWAY_ADDR=\"$address\"|g" .env
```

FINALLY, `source .env` to update the environment variables.

## Fund your accounts

Run the following:

```bash
show_actor_addresses
```

For each one, fund the accounts using the [faucet](https://faucet.testnet.pokt.network/)

Next, run this helper (it's part of `helpers.sh`) to find each of them on the explorer:

```bash
show_explorer_urls
```

## Stake a Supplier & Deploy a RelayMiner

Stake the supplier:

```bash
sed -i -e s/YOUR_NODE_IP_OR_HOST/$NODE_HOSTNAME/g ./stake_configs/supplier_stake_config_example.yaml
sed -i -e s/YOUR_OWNER_ADDRESS/$SUPPLIER_ADDR/g ./stake_configs/supplier_stake_config_example.yaml
poktrolld tx supplier stake-supplier --config=/poktroll/stake_configs/supplier_stake_config_example.yaml --from=supplier --chain-id=poktroll --yes

# OPTIONALLY check the supplier's status
poktrolld query supplier show-supplier $SUPPLIER_ADDR

# Start the relay miner (please update the grove app ID if you can)
sed -i -e s/YOUR_NODE_IP_OR_HOST/$NODE_HOSTNAME/g relayminer/config/relayminer_config.yaml
sed -i -e "s|backend_url: \".*\"|backend_url: \"https://eth-mainnet.rpc.grove.city/v1/c7f14c60\"|g" relayminer/config/relayminer_config.yaml
```

Start the supplier

```bash
docker compose up -d relayminer
# OPTIONALLY view the logs
docker logs -f --tail 100 relayminer
```

## Stake an Application & Deploy an AppGate Server

Stake the application:

```bash
poktrolld tx application stake-application --config=/poktroll/stake_configs/application_stake_config_example.yaml --from=application --chain-id=poktroll --yes

# OPTIONALLY check the application's status
poktrolld query application show-application $APPLICATION_ADDR
```

Start the appgate server:

```bash
docker compose up -d appgate
# OPTIONALLY view the logs
docker logs -f --tail 100 appgate
```

## Send a Relay

```bash
curl http://$NODE_HOSTNAME:85/0021 \
  -X POST \
  -H "Content-Type: application/json" \
  --data '{"method":"eth_blockNumber","params":[],"id":1,"jsonrpc":"2.0"}'
```

### Ensure you get a response

To ensure you get a response, run the request a few times.

```bash
for i in {1..10}; do
  curl http://$NODE_HOSTNAME:85/0021 \
    -X POST \
    -H "Content-Type: application/json" \
    --data '{"method":"eth_blockNumber","params":[],"id":1,"jsonrpc":"2.0"}' \
    --max-time 1
  echo ""
done
```

## [BONUS] Deploy a PATH Gateway

If you want to deploy a real Gateway, you can use [Grove's PATH](https://github.com/buildwithgrove/path)
after running the following commands:

```bash
# Stake the gateway
poktrolld tx gateway stake-gateway --config=/poktroll/stake_configs/gateway_stake_config_example.yaml --from=gateway --chain-id=poktroll --yes
# Delegate from the application to the gateway
poktrolld tx application delegate-to-gateway $GATEWAY_ADDR --from=application --chain-id=poktroll --chain-id=poktroll --yes

# OPTIONALLY check the gateway's and application's status
poktrolld query gateway show-gateway $GATEWAY_ADDR
poktrolld query application show-application $APPLICATION_ADDR
```

## Managing a re-genesis

Assuming you already had everything functioning following the steps above, this
is a quick way to reset everything (without recreating keys) after a re-genesis.

### Full Nodes

```bash
# Stop all containers
docker compose down
docker rm $(docker ps -aq) -f

# Remove existing data
rm -rf poktrolld-data/config/addrbook.json poktrolld-data/config/genesis.json poktrolld-data/config/genesis.seeds poktrolld-data/data/ poktrolld-data/config/node_key.json poktrolld-data/config/priv_validator_key.json
```

Update `POKTROLLD_IMAGE_TAG` in `.env` based on the releases [here](https://github.com/pokt-network/poktroll/releases).

```bash
# Start the full
docker compose up -d full-node

# Sanity check the logs
docker logs full-node -f --tail 100
```

### Fund the same accounts

Go to the [faucet](https://faucet.testnet.pokt.network/) and fund the same accounts:

```bash
echo $APPLICATION_ADDR
echo $GATEWAY_ADDR
echo $SUPPLIER_ADDR
```

#### Faucet is not ready and you need to fund the accounts manually

```bash
# Import the faucet using the mnemonic
poktrolld keys add --recover -i faucet
poktrolld tx bank multi-send faucet $APPLICATION_ADDR $GATEWAY_ADDR $SUPPLIER_ADDR 100000upokt --chain-id=poktroll --yes
```

### Start the RelayMiner

```bash
# Stake
poktrolld tx supplier stake-supplier --config=/poktroll/stake_configs/supplier_stake_config_example.yaml --from=supplier --chain-id=poktroll --yes
# Check
poktrolld query supplier show-supplier $SUPPLIER_ADDR
# Start
docker compose up -d relayminer
# View
docker logs -f --tail 100 relayminer
```

### Start the AppGate Server

```bash
# Stake
poktrolld tx application stake-application --config=/poktroll/stake_configs/application_stake_config_example.yaml --from=application --chain-id=poktroll --yes
# Check
poktrolld query application show-application $APPLICATION_ADDR
# Start
docker compose up -d appgate
# View
docker logs -f --tail 100 appgate
```

### Re-stake the gateway
