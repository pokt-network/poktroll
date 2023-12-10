---
sidebar_position: 2
title: Quickstart
---

# Quickstart <!-- omit in toc -->

- [Report issues](#report-issues)
- [Install Dependencies](#install-dependencies)
- [Launch LocalNet](#launch-localnet)
  - [Clone the repository](#clone-the-repository)
  - [Prepare your environment](#prepare-your-environment)
  - [Create a k8s cluster](#create-a-k8s-cluster)
  - [Start the LocalNet](#start-the-localnet)
- [Interact with the chain](#interact-with-the-chain)
  - [Create a new Account](#create-a-new-account)
  - [Fund your account](#fund-your-account)
  - [Send a relay](#send-a-relay)
  - [Stake Shannon as an Application](#stake-shannon-as-an-application)
  - [Send a relay](#send-a-relay-1)
- [Explore the tools](#explore-the-tools)
  - [poktrolld](#poktrolld)
  - [Makefile](#makefile)
  - [Ignite](#ignite)

The goal of this document is to get you up and running with a LocalNet and end-to-end relay.

## Report issues

If you encounter any problems, please create a new [GitHub Issue here](https://github.com/pokt-network/pocket/issues/new/choose).

## Install Dependencies

Install the following:

1. [Golang](https://go.dev/doc/install)
2. [Docker](https://docs.docker.com/get-docker/)
3. [Ignite](https://docs.ignite.com/welcome/install)
4. [Kind](https://kind.sigs.k8s.io/#installation-and-usage)
5. [Helm](https://helm.sh/docs/intro/install/#through-package-managers)
6. [Tilt](https://docs.tilt.dev/install.html)

:::note
You might already have these installed if you've followed the [localnet instructions](./infrastructure/localnet.md).
:::

## Launch LocalNet

### Clone the repository

```bash
git clone https://github.com/pokt-network/poktroll.git
cd poktroll
```

### Prepare your environment

Generate mocks, compile the protobufs and verify that all the tests are passing by running:

```bash
make go_develop_and_test
```

### Create a k8s cluster

```bash
kind create cluster
```

### Start the LocalNet

```bash
make localnet_up
```

Visit [localhost:10350](http://localhost:10350) and wait until all the containers are 🟢.

![LocalNet](./img/quickstart_localnet.png)

## Interact with the chain

### Create a new Account

List all the accounts we get out of the box by running:

```bash
ignite account list --keyring-dir=$(POKTROLLD_HOME) --keyring-backend test --address-prefix $(POCKET_ADDR_PREFIX)
```

And create a new account named `shannon` by running:

```bash
ignite account create shannon --keyring-dir=./localnet/poktrolld --keyring-backend test
```

If you re-run the command above, it should show up in the list.
Make sure to note its address under the `Address` column and export it as an
environment variable for convenience. For example:

```bash
export SHANNON_ADDRESS=pokt1mczm7xste7ckrwrmerda7m5ze89gyd9rzvxztr
```

### Fund your account

Query your account's balance by running:

```bash
poktrolld --home=./localnet/poktrolld q bank balances $SHANNON_ADDRESS --node tcp://127.0.0.1:36657
```

And you should see an empty balance:

```yaml
balances: []
pagination:
  next_key: null
  total: "0"
```

But our sequencer has a lot of pokt from the genesis.json file (found at `localnet/poktrolld/config/genesis.json`)

```bash
poktrolld --home=./localnet/poktrolld tx bank send sequencer1 $SHANNON_ADDRESS 199999100000000upokt --node tcp://127.0.0.1:36657
```

And you'll find that Shannon is now rolling in POKT:

```yaml
balances:
  - amount: "199999100000000"
    denom: upokt
pagination:
  next_key: null
  total: "0"
```

### Send a relay

```bash
curl -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' http://localhost:42069/anvil
```

### Stake Shannon as an Application

Run `make app_list` (a helper our team created) to see all the apps staked on the network.
You should see that `SHANNON_ADDRESS` is not staked as an app yet.

In order to stake shannon as an app, we need to create a new config file and run
the stake command.

```bash
touch shannon_app_config.yaml
cat <<EOF >> shannon_app_config.yaml
service_ids:
 - anvil
EOF
```

We already have a supplier pre-configured to supply services for anvil
(an local ethereum testing node), so we simply reused that for simplicity.

Next, run the stake command:

```bash
poktrolld --home=./localnet/poktrolld tx application stake-application 1000upokt --config shannon_app_config.yaml --keyring-backend test --from shannon --node tcp://127.0.0.1:36657
```

If you re-run, `make app_list` you should see that `SHANNON_ADDRESS` is now staked as an app.

### Send a relay

## Explore the tools

There are three primary tools you'll use to develop and interact with the network:

1. `poktrolld` - the Pocket Rollup Node
2. `make` - a collection of helpers to make your life easier
3. `ignite` - a tool to manage the local k8s cluster

### poktrolld

### Makefile

### Ignite

```

1. Mint som new tokens
2. Stake an application
3. Send some funds
4. Send a relay

## Develop

## Tools

### poktrolld

Run `poktrolld --help`

### Makefile

Run `make` to see all the helpers we're working on

### Ignite

### LocalNe

```

```

```
