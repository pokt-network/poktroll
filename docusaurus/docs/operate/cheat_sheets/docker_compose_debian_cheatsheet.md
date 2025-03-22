---
sidebar_position: 7
title: "[Deprecated] Docker Compose E2E (> 1 hour)"
---

import ReactPlayer from "react-player";

## Docker Compose E2E Cheat Sheet <!-- omit in toc -->

- [Results](#results)
- [Deploy your server](#deploy-your-server)
- [Install Dependencies](#install-dependencies)
- [Retrieve the source code](#retrieve-the-source-code)
- [Update your environment](#update-your-environment)
- [Start up the full node](#start-up-the-full-node)
- [Create new addresses for all your accounts and update .env](#create-new-addresses-for-all-your-accounts-and-update-env)
- [Fund your accounts](#fund-your-accounts)
- [Stake a Supplier \& Deploy a RelayMiner](#stake-a-supplier--deploy-a-relayminer)
- [Stake an Application and Gateway](#stake-an-application-and-gateway)
- [Deploy a PATH Gateway](#deploy-a-path-gateway)
- [Send a Relay](#send-a-relay)
  - [Ensure you get a response](#ensure-you-get-a-response)
- [Managing a re-genesis](#managing-a-re-genesis)
  - [Full Nodes](#full-nodes)
  - [Fund the same accounts](#fund-the-same-accounts)
    - [Faucet is not ready and you need to fund the accounts manually](#faucet-is-not-ready-and-you-need-to-fund-the-accounts-manually)
  - [Start the RelayMiner](#start-the-relayminer)
  - [Start the PATH Gateway](#start-the-path-gateway)

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
    echo "UFW rules added for Docker networks and validator endpoint"
else
    echo "UFW is not installed, skipping firewall configuration"
fi
```

And then install docker:

```bash
sudo apt-get install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
```

### [Optional] Create a new user <!-- omit in toc -->

You can optionally create a new user and give it sudo permissions instead of using `root`.

```bash
adduser poktroll
usermod -aG docker,sudo poktroll
su - poktroll
```

## Retrieve the source code

Then pull the github repo

```bash
mkdir ~/workspace && cd ~/workspace
git clone https://github.com/pokt-network/poktroll-docker-compose-example.git
cd poktroll-docker-compose-example
```

## Update your environment

First, copy the sample environment file:

```bash
cp .env.sample .env
```

By default, the `.env` file uses `testnet-beta`. If you want to use a different network, update the `NETWORK_NAME` in your `.env` file to one of:

- `testnet-alpha`: Unstable testnet (use at your own risk)
- `testnet-beta`: Stable testnet (default)
- `mainnet`: Production network (not launched yet)

Then set your external IP and source the environment:

```bash
EXTERNAL_IP=$(curl -4 ifconfig.me/ip)
sed -i -e s/NODE_HOSTNAME=/NODE_HOSTNAME=$EXTERNAL_IP/g .env

echo "source $(pwd)/helpers.sh" >> ~/.bashrc
echo "source $(pwd)/.env" >> ~/.bashrc
source ~/.bashrc
```

## Start up the full node

:::warning
The Alpha TestNet currently requires manual steps to sync the node to the latest block. Please find the affected block(s)
in [this document](../upgrades/upgrade_list.md), which leads to the manual upgrade instructions.
:::

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
poktrolld keys add application

privKey=$(export_priv_key_hex application); sed -i "s|APPLICATION_PRIV_KEY_HEX=\".*\"|APPLICATION_PRIV_KEY_HEX=\"$privKey\"|" .env

address=$(poktrolld keys show application -a | tr -d '\r'); sed -i "s|APPLICATION_ADDR=\".*\"|APPLICATION_ADDR=\"$address\"|g" .env
```

Gateway:

```bash
poktrolld keys add gateway

privKey=$(export_priv_key_hex gateway); sed -i "s|GATEWAY_PRIV_KEY_HEX=\".*\"|GATEWAY_PRIV_KEY_HEX=\"$privKey\"|" .env

address=$(poktrolld keys show gateway -a | tr -d '\r'); sed -i "s|GATEWAY_ADDR=\".*\"|GATEWAY_ADDR=\"$address\"|g" .env
```

FINALLY, update your environment:

```bash
source .env
```

## Fund your accounts

Run the following helper command to see your addresses:

```bash
show_actor_addresses
```

Get the faucet URL for your network:

```bash
show_faucet_url
```

Fund each address using the faucet URL shown above.
Then run this helper to find each account on the explorer:

```bash
show_explorer_urls
```

## Stake a Supplier & Deploy a RelayMiner

Stake the supplier:

```bash
sed -i -e s/YOUR_NODE_IP_OR_HOST/$NODE_HOSTNAME/g ./stake_configs/supplier_stake_config_example.yaml
sed -i -e s/YOUR_OWNER_ADDRESS/$SUPPLIER_ADDR/g ./stake_configs/supplier_stake_config_example.yaml
poktrolld tx supplier stake-supplier --config=/poktroll/stake_configs/supplier_stake_config_example.yaml --from=supplier $TX_PARAM_FLAGS_BETA

# OPTIONALLY check the supplier's status
poktrolld query supplier show-supplier $SUPPLIER_ADDR

# Start the relay miner (please update the grove app ID if you can)
sudo sed -i -e s/YOUR_NODE_IP_OR_HOST/$NODE_HOSTNAME/g relayminer/config/relayminer_config.yaml
sudo sed -i -e "s|backend_url: \".*\"|backend_url: \"https://eth-mainnet.rpc.grove.city/v1/c7f14c60\"|g" relayminer/config/relayminer_config.yaml
```

Start the supplier

```bash
docker compose up -d relayminer
# OPTIONALLY view the logs
docker logs -f --tail 100 relayminer
```

## Stake an Application and Gateway

Stake the application:

```bash
poktrolld tx application stake-application --config=/poktroll/stake_configs/application_stake_config_example.yaml --from=application $TX_PARAM_FLAGS_BETA

# OPTIONALLY check the application's status
poktrolld query application show-application $APPLICATION_ADDR
```

Stake the gateway:

```bash
poktrolld tx gateway stake-gateway --config=/poktroll/stake_configs/gateway_stake_config_example.yaml --from=gateway $TX_PARAM_FLAGS_BETA

# OPTIONALLY check the application's status
poktrolld query gateway show-gateway $GATEWAY_ADDR
```

Delegate the application to the gateway:

```bash
poktrolld tx application delegate-to-gateway $GATEWAY_ADDR --from=application $TX_PARAM_FLAGS_BETA

# OPTIONALLY check the application's delegation status
poktrolld query application show-application $APPLICATION_ADDR
```

## Deploy a PATH Gateway

Configure the PATH gateway:

```bash
sudo sed -i -e s/YOUR_PATH_GATEWAY_ADDRESS/$GATEWAY_ADDR/g gateway/config/gateway_config.yaml
sudo sed -i -e s/YOUR_PATH_GATEWAY_PRIVATE_KEY/$GATEWAY_PRIV_KEY_HEX/g gateway/config/gateway_config.yaml
sudo sed -i -e s/YOUR_OWNED_APP_PRIVATE_KEY/$APPLICATION_PRIV_KEY_HEX/g gateway/config/gateway_config.yaml
```

Start the PATH gateway:

```bash
docker compose up -d gateway
# OPTIONALLY view the logs
docker logs -f --tail 100 gateway
```

## Send a Relay

```bash
curl http://eth.localhost:3000/v1 \
  -X POST \
  -H "Content-Type: application/json" \
  --data '{"method":"eth_blockNumber","params":[],"id":1,"jsonrpc":"2.0"}'
```

### Ensure you get a response

To ensure you get a response, run the request a few times.

```bash
for i in {1..10}; do
  curl http://eth.localhost:3000/v1 \
    -X POST \
    -H "Content-Type: application/json" \
    --data '{"method":"eth_blockNumber","params":[],"id":1,"jsonrpc":"2.0"}' \
    --max-time 1
  echo ""
done
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
rm -rf poktrolld-data/config/addrbook.json poktrolld-data/config/genesis.json poktrolld-data/config/genesis.seeds poktrolld-data/data/ poktrolld-data/cosmovisor/ poktrolld-data/config/node_key.json poktrolld-data/config/priv_validator_key.json
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
poktrolld tx bank multi-send faucet $APPLICATION_ADDR $GATEWAY_ADDR $SUPPLIER_ADDR 100000upokt $TX_PARAM_FLAGS_BETA
```

### Start the RelayMiner

```bash
# Stake
poktrolld tx supplier stake-supplier --config=/poktroll/stake_configs/supplier_stake_config_example.yaml --from=supplier $TX_PARAM_FLAGS_BETA
# Check
poktrolld query supplier show-supplier $SUPPLIER_ADDR
# Start
docker compose up -d relayminer
# View
docker logs -f --tail 100 relayminer
```

### Start the PATH Gateway

```bash
# Stake
poktrolld tx application stake-application --config=/poktroll/stake_configs/application_stake_config_example.yaml --from=application $TX_PARAM_FLAGS_BETA
# Check
poktrolld query application show-application $APPLICATION_ADDR
# Start
docker compose up -d gateway
# View
docker logs -f --tail 100 gateway
```
