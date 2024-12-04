---
sidebar_position: 5
title: Gateway Cheat Sheet
---

# Gateway Cheat Sheet <!-- omit in toc -->

This guide provides quick reference commands for setting up and running a Gateway
on Pocket Network.

:::warning

These instructions are intended to run on a Linux machine.

TODO_TECHDEBT(@olshansky): Adapt the instructions to be macOS friendly.

:::

- [Pre-Requisites](#pre-requisites)
- [Account Setup](#account-setup)
  - [Create and fund the `Gateway` and `Application` accounts](#create-and-fund-the-gateway-and-application-accounts)
  - [Prepare your environment](#prepare-your-environment)
  - [Fund the Gateway and Application accounts](#fund-the-gateway-and-application-accounts)
  - [Stake the `Gateway`](#stake-the-gateway)
  - [Stake the delegating `Application`](#stake-the-delegating-application)
  - [Delegate the `Application` to the `Gateway`](#delegate-the-application-to-the-gateway)
- [`PATH` Setup](#path-setup)
  - [`PATH` Gateway Setup](#path-gateway-setup)
  - [Generate a `PATH Gateway` config file for the Shannon network](#generate-a-path-gateway-config-file-for-the-shannon-network)
  - [Run the `PATH` Gateway](#run-the-path-gateway)
    - [Build and run the `PATH` Gateway from source](#build-and-run-the-path-gateway-from-source)
    - [\[TODO\] Run the `PATH` Gateway using Docker](#todo-run-the-path-gateway-using-docker)
  - [Check the `PATH Gateway` is serving relays](#check-the-path-gateway-is-serving-relays)

:::note
For detailed instructions, troubleshooting, and observability setup, see the [Gateway Walkthrough](./../run_a_node/gateway_walkthrough.md).
:::

## Pre-Requisites

1. Make sure to [install the `poktrolld` CLI](../user_guide/install.md).
2. Make sure you know how to [create and fund a new account](../user_guide/create-new-wallet.md).

## Account Setup

### Create and fund the `Gateway` and `Application` accounts

Create a new key pair for the delegating `Application`:

```bash
poktrolld keys add application

# Optionally, to avoid entering the password each time:
# poktrolld keys add application --keyring-backend test
```

Create a new key pair for the `Gateway`:

```bash
poktrolld keys add gateway

# Optionally, to avoid entering the password each time:
# poktrolld keys add gateway --keyring-backend test
```

:::tip

You can set the `--keyring-backend` flag to `test` to avoid entering the password
each time. Learn more about [cosmos keyring backends here](https://docs.cosmos.network/v0.46/run-node/keyring.html).

:::

### Prepare your environment

For convenience, we're setting several environment variables to streamline
the process of interacting with the Shannon network:

We recommend you put these in your `~/.bashrc` file:

```bash
export NODE="https://shannon-testnet-grove-rpc.beta.poktroll.com"
export NODE_FLAGS="--node=https://shannon-testnet-grove-rpc.beta.poktroll.com"
export TX_PARAM_FLAGS="--gas=auto --gas-prices=1upokt --gas-adjustment=1.5 --chain-id=pocket-beta --yes"
export GATEWAY_ADDR=$(poktrolld keys show gateway -a)
export APP_ADDR=$(poktrolld keys show application -a)

# Optionally, to avoid entering the password each time:
# export GATEWAY_ADDR=$(poktrolld keys show gateway -a --keyring-backend test)
# export APP_ADDR=$(poktrolld keys show application -a --keyring-backend test)
```

:::tip

You can put the above in a special `~/.pocketrc` and add `source ~/.pocketrc` to
your `~/.bashrc` file for a cleaner organization.

:::

### Fund the Gateway and Application accounts

Run the following command to get the `Gateway` and `Application` addresses:

```bash
echo "Gateway address: $GATEWAY_ADDR"
echo "Application address: $APP_ADDR"
```

Then use the [Shannon Beta TestNet faucet](https://faucet.beta.testnet.pokt.network/) to fund the `Gateway`
and `Application` accounts respectively.

Afterwards, you can query their balances using the following command:

```bash
poktrolld query bank balances $GATEWAY_ADDR $NODE_FLAGS
poktrolld query bank balances $APP_ADDR $NODE_FLAGS
```

:::tip
You can find all the explorers, faucets and tools at the [tools page](../../explore/tools.md).
:::

### Stake the `Gateway`

Create a Gateway stake configuration file:

```bash
cat <<EOF > /tmp/stake_gateway_config.yaml
stake_amount: 1000000upokt
EOF
```

And run the following command to stake the `Gateway`:

```bash
poktrolld tx gateway stake-gateway --config=/tmp/stake_gateway_config.yaml --from=$GATEWAY_ADDR $TX_PARAM_FLAGS $NODE_FLAGS

# Optionally, to avoid entering the password each time:
# poktrolld tx gateway stake-gateway --config=/tmp/stake_gateway_config.yaml --from=$GATEWAY_ADDR $TX_PARAM_FLAGS $NODE_FLAGS --keyring-backend test
```

After about a minute, you can check the `Gateway`'s status like so:

```bash
poktrolld query gateway show-gateway $GATEWAY_ADDR $NODE_FLAGS
```

### Stake the delegating `Application`

Create an Application stake configuration file:

```bash
cat <<EOF > /tmp/stake_app_config.yaml
stake_amount: 100000000upokt
service_ids:
  - "F00C"
EOF
```

And run the following command to stake the `Application`:

```bash
poktrolld tx application stake-application --config=/tmp/stake_app_config.yaml --from=$APP_ADDR $TX_PARAM_FLAGS $NODE_FLAGS

# Optionally, to avoid entering the password each time:
# poktrolld tx application stake-application --config=/tmp/stake_app_config.yaml --from=$APP_ADDR $TX_PARAM_FLAGS $NODE_FLAGS --keyring-backend test
```

After about a minute, you can check the `Application`'s status like so:

```bash
poktrolld query application show-application $APP_ADDR $NODE_FLAGS
```

### Delegate the `Application` to the `Gateway`

```bash
poktrolld tx application delegate-to-gateway $GATEWAY_ADDR --from=$APP_ADDR $TX_PARAM_FLAGS $NODE_FLAGS

# Optionally, to avoid entering the password each time:
# poktrolld tx application delegate-to-gateway $GATEWAY_ADDR --from=$APP_ADDR $TX_PARAM_FLAGS $NODE_FLAGS --keyring-backend test
```

After about a minute, you can check the `Application`'s status like so:

```bash
poktrolld query application show-application $APP_ADDR $NODE_FLAGS
```

## `PATH` Setup

### `PATH` Gateway Setup

Assuming you have followed the instructions above, the following should be true:

1. You have created, funded and stake a `Gateway`.
2. You have created, funded and stake a `Application`.
3. You have from the staked `Application` to staked the `Gateway`.

Next, you can run a `PATH` Gateway.

Star by following these instructions:

```bash
cd ~/workspace
git clone https://github.com/buildwithgrove/path.git
cd path
```

### Generate a `PATH Gateway` config file for the Shannon network

<!-- TODO_MAINNET(red-0ne): Link to PATH Gateway modes documentation once available -->

:::note

The instructions below show how to setup a `PATH` in `Centralized Mode` (i.e. The operator owns
both the `Gateway` and the `Application` accounts).

Refer to [PATH Gateway modes](https://path.grove.city/) for more configuration options.

:::

Run the following command to generate a default Shannon config `cmd/.config.yaml`:

_NOTE: You'll be prompted to confirm the `gateway` account private key export._

```bash
# Make a copy of the default config file
make copy_shannon_config

# Replace the endpoints as needed
sed -i "s|rpc_url: ".*"|rpc_url: $NODE|" cmd/.config.yaml
sed -i "s|host_port: ".*"|host_port: shannon-testnet-grove-grpc.beta.poktroll.com:443|" cmd/.config.yaml

# Update the gateway and application addresses
sed -i "s|gateway_address: .*|gateway_address: $GATEWAY_ADDR|" cmd/.config.yaml
sed -i "s|gateway_private_key_hex: .*|gateway_private_key_hex: $(poktrolld keys export gateway --unsafe --unarmored-hex)|" cmd/.config.yaml
sed -i '/owned_apps_private_keys_hex:/!b;n;c\      - '"$(poktrolld keys export application --unsafe --unarmored-hex)" cmd/.config.yaml

# If you're using the test keyring-backend:
# sed -i "s|gateway_private_key_hex: .*|gateway_private_key_hex: $(poktrolld keys export gateway --unsafe --unarmored-hex --keyring-backend test)|" cmd/.config.yaml
# sed -i '/owned_apps_private_keys_hex:/!b;n;c\      - '"$(poktrolld keys export application --unsafe --unarmored-hex --keyring-backend test)" cmd/.config.yaml
```

When you're done, run `cat cmd/.config.yaml` to view the updated config file.

### Run the `PATH` Gateway

#### Build and run the `PATH` Gateway from source

```bash
cd cmd/ && go build -o path . && ./path
```

You should see the following output:

```json
{"level":"info","message":"Starting the cache update process."}
{"level":"warn","message":"endpoint hydrator is disabled: no service QoS generators are specified"}
{"level":"info","package":"router","message":"PATH gateway running on port 3000"}
```

#### [TODO] Run the `PATH` Gateway using Docker

_TODO_IMPROVE(@olshansk): Add instructions for running the `PATH` Gateway using Docker._

### Check the `PATH Gateway` is serving relays

Check that the `PATH Gateway` is serving relays by running the following command yourself:

```bash
curl http://eth.localhost:3000/v1 \
    -X POST \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc": "2.0", "id": 1, "method": "eth_blockNumber" }'
```

:::warning

Requests MAY hit unresponsive nodes. If that happens, keep retrying the request a few times.

Once `PATH`s QoS module is mature, this will be handled automatically.

:::
