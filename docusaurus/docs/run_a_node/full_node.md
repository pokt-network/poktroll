---
title: Full Node - Docker Compose
---

- [What is a Full Node?](#what-is-a-full-node)
- [0. Prerequisites](#0-prerequisites)
- [1. Clone the Repository](#1-clone-the-repository)
- [2. Download Network Genesis](#2-download-network-genesis)
- [3. Configure Environment Variables](#3-configure-environment-variables)
- [4. Launch the Node](#4-launch-the-node)


### What is a Full Node?

A full node in a blockchain network maintains a complete copy of the ledger, verifying transactions and blocks against the network's rules without participating in block creation or consensus. It ensures data accuracy, supports network security, and promotes decentralization by relaying transactions and blocks to other nodes.

Within the Pocket Network ecosystem, Full Nodes are especially important for Node Runners. This is because our off-chain actors, such as RelayMiners and AppGates, require communication with the Pocket Network blockchain to function properly.

This guide will demonstrate the process of setting up a Full Node using Docker Compose. This method offers a straightforward and efficient approach to initiate a full node operation.

### 0. Prerequisites

Ensure the following software is installed on your system:
- [git](https://github.com/git-guides/install-git);
- [Docker](https://docs.docker.com/engine/install/);
- [docker-compose](https://docs.docker.com/compose/install/#installation-scenarios);

Additionally, the system and network setup must be capable of exposing ports to the internet for peer-to-peer communication.

### 1. Clone the Repository

```
git clone https://github.com/pokt-network/poktroll-docker-compose-example.git
cd poktroll-docker-compose-example
```

### 2. Download Network Genesis

The Poktrolld blockchain deploys various networks (e.g., testnets, mainnet). Access the list of Poktrolld networks available for community participation here: [Poktrolld Networks](https://github.com/pokt-network/pocket-network-genesis/tree/master/poktrolld).

Download and place the genesis.json for your chosen network (e.g., testnet-validated) into the poktrolld/config directory:

```bash
NETWORK_NAME=testnet-validated curl https://raw.githubusercontent.com/pokt-network/pocket-network-genesis/master/poktrolld/${NETWORK_NAME}.json > poktrolld-data/config/genesis.json
```

### 3. Configure Environment Variables

Create and configure your `.env` file from the sample:

```bash
cp .env.sample .env
```

Update `NODE_HOSTNAME` in `.env` to the IP address or hostname of your node.

### 4. Launch the Node

Initiate the node with:

```bash
docker-compose up -d
```

Monitor node activity through logs with:

```bash
docker-compose logs -f
```

