---
sidebar_position: 5
title: Gateway Cheat Sheet
---

## Gateway Cheat Sheet <!-- omit in toc -->
<!-- markdownlint-disable MD014 -->

:::tip

See the [Gateway Walkthrough](./../run_a_node/gateway_walkthrough.md) for an in-depth guide on setting up a Gateway, troubleshooting, observability and more.

:::

## Build Shannon binary

The `poktrolld` binary is needed to stake your `Gateway` and its delegating `Application`s.

**Retrieve source code and build binaries**

```bash
mkdir ~/poktroll && cd ~/poktroll
git clone https://github.com/pokt-network/poktroll.git
cd poktroll
make ignite_poktrolld_build
```

## Create, fund and stake the Gateway and Application accounts

**Create a new key pair for the delegating `Application`**

```bash
poktrolld keys add application
```

**Create a new key pair for the `Gateway`**

```bash
poktrolld keys add gateway
```

**Fund the Gateway and Application accounts**

Retrieve the `Gateway` and `Application` addresses:

```bash
echo "Gateway address: $(poktrolld keys show gateway -a)"
echo "Application address: $(poktrolld keys show application -a)"
```

Then use the [faucet](https://faucet.testnet.pokt.network/) to fund the `Gateway`
and `Application` accounts.

**Stake the Gateway**

```bash
# Create a Gateway stake configuration file
cat <<EOF > /tmp/stake_gateway
stake_amount: 1000000upokt
EOF

poktrolld tx gateway stake-gateway --config=/tmp/stake_gateway --from=gateway --chain-id=poktroll --yes

# OPTIONALLY check the gateway's status
poktrolld query gateway show-gateway $(poktrolld keys show gateway -a)
```

**Stake the delegating Application**

```bash
# Create an Application stake configuration file
cat <<EOF > /tmp/stake_app
stake_amount: 10000000upokt
service_ids:
  - "0021"
EOF

poktrolld tx application stake-application --config=/tmp/stake_app --from=application --chain-id=poktroll --yes

# OPTIONALLY check the application's status
poktrolld query application show-application $(poktrolld keys show application -a)
```

**Delegate the Application to the Gateway**

```bash
poktrolld tx application delegate-to-gateway $(poktrolld keys show gateway -a) --from=application --chain-id=poktroll --chain-id=poktroll --yes

# OPTIONALLY check the application's status
poktrolld query application show-application $(poktrolld keys show application -a)
```

## Retrieve PATH Gateway source code

Pull the latest `PATH Gateway` source code:

```bash
mkdir ~/path-gateway && cd ~/path-gateway
git clone https://github.com/buildwithgrove/path.git
cd path-gateway
```

## Generate a PATH Gateway config file for the Shannon network

Run the following command to generate a default Shannon config `cmd/.config.yaml`

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

**Update the `cmd/.config.yaml` file with the `Gateway` and `Application` keys**

```bash
sed -i '/owned_apps_private_keys_hex:/!b;n;c\      - '"$(yes | poktrolld keys export application --unsafe --unarmored-hex)" cmd/.config.yaml
sed -i "s|gateway_address: .*|gateway_address: $(poktrolld keys show gateway -a)|" cmd/.config.yaml
sed -i "s|gateway_private_key_hex: .*|gateway_private_key_hex: $(yes | poktrolld keys export gateway --unsafe --unarmored-hex)|" cmd/.config.yaml
```

## Build and run the PATH Gateway

```bash
cd cmd/ && go build -o path . && ./path
```

You should see the following output:

```json
{"level":"info","message":"Starting the cache update process."}
{"level":"warn","message":"endpoint hydrator is disabled: no service QoS generators are specified"}
{"level":"info","package":"router","message":"PATH gateway running on port 3000"}
{"level":"warn","error":"buildAppsServiceMap: no apps found.","message":"updateAppCache: error getting the list of apps; skipping update."}
{"level":"warn","method":"fetchSessions","error":"buildAppsServiceMap: no apps found.","message":"fetchSession: error listing applications"}
{"level":"warn","message":"updateSessionCache: received empty session list; skipping update."}
```

Check that the `PATH Gateway` is serving relays:

```bash
curl http://eth.localhost:3000/v1 \
    -X POST \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc": "2.0", "id": 1, "method": "eth_blockNumber" }'
```