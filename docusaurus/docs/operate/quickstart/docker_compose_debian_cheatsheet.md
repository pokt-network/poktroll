---
sidebar_position: 2
title: Docker Compose - Debian Cheatsheet
---

# tl;dr Debian Cheat Sheet <!-- omit in toc -->

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
```

And then install docker:

```bash
sudo apt-get install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
```

## Retrieve the source code

Then install docker-compose

```bash
mkdir ~/workspace && cd ~/workspace
git clone https://github.com/pokt-network/poktroll-docker-compose-example.git
cd poktroll-docker-compose-example
```

## Update your environment

```bash
echo "source $(pwd)/helpers.sh" >> ~/.bashrc
source ~/.bashrc

EXTERNAL_IP=$(curl -4 ifconfig.me/ip)

cp .env.sample .env
sed -i -e s/NODE_HOSTNAME=/NODE_HOSTNAME=$EXTERNAL_IP/g .env
echo "source $(pwd)/.env" >> ~/.bashrc
source ~/.bashrc
```

## Start up the full node

```bash
docker compose up -d poktrolld poktrolld
# Optional: watch the block height sync up & logs
docker logs -f --tail 100 full_node
watch_height
```

## Create new addresses for all your accounts and update .env

Supplier:

```bash
poktrolld keys add supplier > /tmp/supplier

mnemonic=$(tail -n 1 /tmp/supplier | tr -d '\r') sed -i "s|SUPPLIER_MNEMONIC=\".*\"|SUPPLIER_MNEMONIC=\"$mnemonic\"|" .env

address=$(awk '/address:/{print $3; exit}' /tmp/supplier | tr -d '\r'); sed -i "s|SUPPLIER_ADDR=\".*\"|SUPPLIER_ADDR=\"$address\"|g" .env
```

Application:

```bash
poktrolld keys add application > /tmp/application

mnemonic=$(tail -n 1 /tmp/application | tr -d '\r') sed -i "s|APPLICATION_MNEMONIC=\".*\"|APPLICATION_MNEMONIC=\"$mnemonic\"|" .env

address=$(awk '/address:/{print $3; exit}' /tmp/application | tr -d '\r'); sed -i "s|APPLICATION_ADDR=\".*\"|APPLICATION_ADDR=\"$address\"|g" .env
```

Gateway:

```bash
poktrolld keys add gateway > /tmp/gateway

mnemonic=$(tail -n 1 /tmp/gateway | tr -d '\r') sed -i "s|GATEWAY_MNEMONIC=\".*\"|GATEWAY_MNEMONIC=\"$mnemonic\"|" .env

address=$(awk '/address:/{print $3; exit}' /tmp/gateway | tr -d '\r'); sed -i "s|GATEWAY_ADDR=\".*\"|GATEWAY_ADDR=\"$address\"|g" .env
```

Finally, `source .env` to update the environment variables.

## Fund your accounts

Run the following:

```bash
echo $APPLICATION_ADDR
echo $GATEWAY_ADDR
echo $SUPPLIER_ADDR
```

For each one, fund the accounts using the [faucet](https://faucet.testnet.pokt.network/)

Next, run this helper (it's part of `helpers.sh`) to find each of them on the explorer:

```bash
explorer_urls
```

## Stake a Supplier & Deploy a RelayMiner

```bash
# Stake the supplier
sed -i -e s/YOUR_NODE_IP_OR_HOST/$NODE_HOSTNAME/g ./stake_configs/supplier_stake_config_example.yaml
poktrolld tx supplier stake-supplier --config=/poktroll/stake_configs/supplier_stake_config_example.yaml --from=supplier --chain-id=poktroll --yes
# OPTIONALLY check the supplier's status
poktrolld query supplier show-supplier $SUPPLIER_ADDR

# Start the relay miner (please update the grove app ID if you can)
sed -i -e s/YOUR_NODE_IP_OR_HOST/$NODE_HOSTNAME/g relayminer-example/config/relayminer_config.yaml
sed -i -e "s|backend_url: \".*\"|backend_url: \"https://eth-mainnet.rpc.grove.city/v1/c7f14c60\"|g" relayminer-example/config/relayminer_config.yaml
sed -i -e s/key-for-supplier1/supplier/g relayminer-example/config/relayminer_config.yaml
docker-compose up -d relayminer-example
# OPTIONALLY view the logs
docker logs -f --tail 100 relay_miner
```

## Stake an Application & Deploy an AppGate Server

```bash
# Stake the application
poktrolld tx application stake-application --config=/poktroll/stake_configs/application_stake_config_example.yaml --from=application --chain-id=poktroll --yes
# OPTIONALLY check the application's status
poktrolld query application show-application $APPLICATION_ADDR

# Start the appgate server
docker compose up -d appgate-server-example
# OPTIONALLY view the logs
docker logs -f --tail 100 appgate_server
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
