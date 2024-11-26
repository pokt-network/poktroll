---
sidebar_position: 5
title: Gateway Cheat Sheet
---

# Gateway Cheat Sheet <!-- omit in toc -->

This guide provides quick reference commands for setting up and running a gateway node.

- [Prerequisites](#prerequisites)
- [Build Pocket](#build-pocket)
- [Account Setup](#account-setup)
  - [Create and fund the `Gateway` and `Application` accounts](#create-and-fund-the-gateway-and-application-accounts)
  - [Fund the Gateway and Application accounts](#fund-the-gateway-and-application-accounts)
  - [Stake the `Gateway`](#stake-the-gateway)
  - [Stake the delegating `Application`](#stake-the-delegating-application)
  - [Delegate the `Application` to the `Gateway`](#delegate-the-application-to-the-gateway)
- [`PATH Gateway` Configuration](#path-gateway-configuration)
  - [Generate a `PATH Gateway` config file for the Shannon network](#generate-a-path-gateway-config-file-for-the-shannon-network)
- [`PATH Gateway` Setup](#path-gateway-setup)
  - [Build and run the PATH Gateway](#build-and-run-the-path-gateway)

:::tip
For detailed instructions, troubleshooting, and observability setup, see the [Gateway Walkthrough](./../run_a_node/gateway_walkthrough.md).
:::

## Prerequisites

Install the required dependencies:

```bash
# Install go 1.23
curl -o ./pkgx --compressed -f --proto '=https' https://pkgx.sh/$(uname)/$(uname -m)
sudo install -m 755 pkgx /usr/local/bin
pkgx install go@1.23.0
export PATH=$PATH:$HOME/go/bin/

# Install PATH Gateway required dependencies
apt-get update && apt-get install git make build-essential

# Install the ignite binary used to build the Pocket binary
curl https://get.ignite.com/cli! | bash
```

## Build Pocket

The `poktrolld` binary is needed to stake your `Gateway` and its delegating `Application`s.

Retrieve source code and build binaries:

```bash
git clone https://github.com/pokt-network/poktroll.git
cd poktroll
make ignite_poktrolld_build
```

## Account Setup

### Create and fund the `Gateway` and `Application` accounts

Create a new key pair for the delegating `Application`:

```bash
poktrolld keys add application
```

Create a new key pair for the `Gateway`:

```bash
poktrolld keys add gateway
```

### Fund the Gateway and Application accounts

Set the environment variables needed to interact with the Shannon network:

```bash
export NODE="--node=https://shannon-testnet-grove-rpc.beta.poktroll.com"
export TX_PARAMS="--gas=auto --gas-prices=1upokt --gas-adjustment=1.5 --chain-id=pocket-beta --yes"
export GATEWAY_ADDR=$(poktrolld keys show gateway -a)
export APP_ADDR=$(poktrolld keys show application -a)

echo "Gateway address: $GATEWAY_ADDR"
echo "Application address: $APP_ADDR"
```

Then use the [faucet](https://faucet.beta.testnet.pokt.network/) to fund the `Gateway`
and `Application` accounts.

### Stake the `Gateway`

```bash
# Create a Gateway stake configuration file
cat <<EOF > /tmp/stake_gateway
stake_amount: 1000000upokt
EOF

poktrolld tx gateway stake-gateway --config=/tmp/stake_gateway --from=$GATEWAY_ADDR $TX_PARAMS $NODE
```

Optionally check the `Gateway`'s status:
```bash
poktrolld query gateway show-gateway $GATEWAY_ADDR $NODE
```

### Stake the delegating `Application`

```bash
# Create an Application stake configuration file
cat <<EOF > /tmp/stake_app
stake_amount: 100000000upokt
service_ids:
  - "0021"
EOF

poktrolld tx application stake-application --config=/tmp/stake_app --from=$APP_ADDR $GAS_PARAMS $NODE
```

Optionally check the `Application`'s status:
```bash
poktrolld query application show-application $APP_ADDR $NODE
```

### Delegate the `Application` to the `Gateway`

```bash
poktrolld tx application delegate-to-gateway $GATEWAY_ADDR --from=$APP_ADDR $GAS_PARAMS $NODE
```

Optionally check if the application is delegating to the gateway
```bash
poktrolld query application show-application $APP_ADDR $NODE
```

## `PATH Gateway` Configuration

Pull the latest `PATH Gateway` source code:

```bash
cd ~
git clone https://github.com/buildwithgrove/path.git
cd path
```

### Generate a `PATH Gateway` config file for the Shannon network

Run the following command to generate a default Shannon config `cmd/.config.yaml`:
<!-- TODO_CONSIDERATION: yq might provide better readability for editing the config file -->

```bash
make copy_shannon_config
sed -i "s|rpc_url: ".*"|rpc_url: https://shannon-testnet-grove-rpc.beta.poktroll.com|" cmd/.config.yaml
sed -i "s|host_port: ".*"|host_port: shannon-testnet-grove-grpc.beta.poktroll.com:443|" cmd/.config.yaml
```

:::note

We aim to setup a `PATH Gateway` with `Centralized Mode` (i.e. The operator owns
both the `Gateway` and the `Application` accounts).

Refer to [PATH Gateway modes](https://path.grove.city/) for more information.
<!-- TODO_BETA(red-0ne): Link to PATH Gateway modes documentation once available -->

:::

Update the `cmd/.config.yaml` file with the `Gateway` address and private key:

_You'll be prompted to confirm the `gateway` account private key export._

```bash
sed -i "s|gateway_address: .*|gateway_address: $GATEWAY_ADDR|" cmd/.config.yaml
sed -i "s|gateway_private_key_hex: .*|gateway_private_key_hex: $(poktrolld keys export gateway --unsafe --unarmored-hex)|" cmd/.config.yaml
```

Update the `cmd/.config.yaml` file with the delegating `Application` private key:

_You'll be prompted to confirm the `application` account private key export._

```bash
sed -i '/owned_apps_private_keys_hex:/!b;n;c\      - '"$(poktrolld keys export application --unsafe --unarmored-hex)" cmd/.config.yaml
```

## `PATH Gateway` Setup

### Build and run the PATH Gateway

```bash
cd cmd/ && go build -o path . && ./path
```

You should see the following output:

```json
{"level":"info","message":"Starting the cache update process."}
{"level":"warn","message":"endpoint hydrator is disabled: no service QoS generators are specified"}
{"level":"info","package":"router","message":"PATH gateway running on port 3000"}
```

Check that the `PATH Gateway` is serving relays:

:::info

Initial requests may hit unresponsive nodes, if that happens, retry the request a few times.
`PATH Gateway` will exclude unresponsive nodes on subsequent requests.

:::

```bash
curl http://eth.localhost:3000/v1 \
    -X POST \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc": "2.0", "id": 1, "method": "eth_blockNumber" }'
```