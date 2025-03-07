---
title: Full Node Walkthrough
sidebar_position: 1
---

import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

**🧑‍🔬 detailed step-by-step instructions to get you up and running with a `Full Node` on Pocket Network ✅**

:::warning This is an in-depth walkthrough

See the [Full Node Cheat Sheet](../cheat_sheets/full_node_cheatsheet.md) if you want to just copy-pasta a few commands.

:::

---

## Table of Contents <!-- omit in toc -->

- [Introduction - why run a Full Node?](#introduction---why-run-a-full-node)
- [Pre-Requisites \& Requirements](#pre-requisites--requirements)
- [Instructions](#instructions)
  - [1. Install Dependencies](#1-install-dependencies)
  - [2. Create a New User](#2-create-a-new-user)
  - [3. Set Up Environment Variables for Cosmovisor](#3-set-up-environment-variables-for-cosmovisor)
  - [4. Install Cosmovisor](#4-install-cosmovisor)
  - [5. Retrieve the Latest Genesis File](#5-retrieve-the-latest-genesis-file)
  - [6. Install `poktrolld`](#6-install-poktrolld)
  - [7. Network Configuration](#7-network-configuration)
  - [8. Sync Options: Genesis vs Snapshot](#8-sync-options-genesis-vs-snapshot)
    - [Option 1: Sync from Genesis](#option-1-sync-from-genesis)
    - [Option 2: Sync from Snapshot (Faster)](#option-2-sync-from-snapshot-faster)
  - [9. Set Up `systemd` Service](#9-set-up-systemd-service)
  - [10. Configure your Firewall](#10-configure-your-firewall)
  - [11. Monitor Your Node](#11-monitor-your-node)

## Introduction - why run a Full Node?

This guide will walk you through, step-by-step, running a Full Node for Pocket Network.

Running a Full Node is the first step toward becoming a Validator, Supplier, or Gateway.

The instructions outlined here use [Cosmovisor](https://docs.cosmos.network/v0.45/run-node/cosmovisor.html)
to enable automatic binary upgrades.

## Pre-Requisites & Requirements

1. **Linux-based System**: Preferably Debian-based distributions (Ubuntu, Debian).
2. **Hardware Requirements**: 
   - 4+ CPU cores
   - 8+ GB RAM
   - 200+ GB SSD storage (for chain data)
3. **Architecture Support**: Both x86_64 (amd64) and ARM64 architectures are supported.
4. **Root or Sudo Access**: Administrative privileges are required.
5. **Dedicated Server or Virtual Machine**: Any provider is acceptable.

## Instructions

### 1. Install Dependencies

Update your package list and install necessary dependencies:

```bash
sudo apt-get update
sudo apt-get install -y curl tar wget jq zstd aria2
```

> **Note:** `zstd` is required for snapshot compression/decompression, and `aria2` is needed for efficient torrent downloads if you choose to sync from a snapshot.

### 2. Create a New User

Create a dedicated user to run `poktrolld`:

```bash
sudo adduser poktroll
```

Set a password when prompted, and add the user to the sudo group:

```bash
sudo usermod -aG sudo poktroll
```

And switch to the `poktroll` user:

```bash
sudo su - poktroll
```

### 3. Set Up Environment Variables for Cosmovisor

Create a `.poktrollrc` file and set the following environment variables:

```bash
touch ~/.poktrollrc

echo "export DAEMON_NAME=poktrolld" >> ~/.poktrollrc
echo "export DAEMON_HOME=\$HOME/.poktroll" >> ~/.poktrollrc
echo "export DAEMON_RESTART_AFTER_UPGRADE=true" >> ~/.poktrollrc
echo "export DAEMON_ALLOW_DOWNLOAD_BINARIES=true" >> ~/.poktrollrc
echo "export UNSAFE_SKIP_BACKUP=false" >> ~/.poktrollrc

echo "source ~/.poktrollrc" >> ~/.profile
source ~/.profile
```

### 4. Install Cosmovisor

Cosmovisor manages the binary upgrades for your node. There are two options to install it:

**Option 1**: Follow the official Cosmovisor installation instructions [here](https://docs.cosmos.network/main/build/tooling/cosmovisor#installation).

**Option 2**: Use the commands below to download and install Cosmovisor:

```bash
mkdir -p $HOME/.local/bin
COSMOVISOR_VERSION="v1.6.0"

ARCH=$(uname -m)
if [ "$ARCH" = "x86_64" ]; then
    ARCH="amd64"
elif [ "$ARCH" = "aarch64" ]; then
    ARCH="arm64"
fi

curl -L "https://github.com/cosmos/cosmos-sdk/releases/download/cosmovisor%2F${COSMOVISOR_VERSION}/cosmovisor-${COSMOVISOR_VERSION}-linux-${ARCH}.tar.gz" | tar -zxvf - -C $HOME/.local/bin

echo 'export PATH=$HOME/.local/bin:$PATH' >> ~/.profile
source ~/.profile
```

### 5. Retrieve the Latest Genesis File

Genesis files and network configuration are stored in the [pocket-network-genesis](https://github.com/pokt-network/pocket-network-genesis) repository. This repository contains the official chain information for all Pocket Network chains.

Choose a network to join from the tabs below:

<Tabs groupId="network">
  <TabItem value="testnet-beta" label="Testnet Beta" default>
    ```bash
    # Set network to testnet-beta (recommended for most users)
    NETWORK="testnet-beta"
    
    # Create config directory if it doesn't exist
    mkdir -p $HOME/.poktroll/config
    
    # Download genesis file
    GENESIS_URL="https://raw.githubusercontent.com/pokt-network/pocket-network-genesis/master/shannon/${NETWORK}/genesis.json"
    curl -s -o $HOME/.poktroll/config/genesis.json "$GENESIS_URL"
    
    # Extract required version from genesis file
    POKTROLLD_VERSION=$(jq -r '.app_version' < $HOME/.poktroll/config/genesis.json)
    echo "Required poktrolld version for testnet-beta: $POKTROLLD_VERSION"
    ```
    
    > **Note:** Testnet Beta is the recommended network for most users. It's more stable than Testnet Alpha and is used for testing features before they reach mainnet.
  </TabItem>
  
  <TabItem value="testnet-alpha" label="Testnet Alpha">
    ```bash
    # Set network to testnet-alpha (unstable testing network)
    NETWORK="testnet-alpha"
    
    # Create config directory if it doesn't exist
    mkdir -p $HOME/.poktroll/config
    
    # Download genesis file
    GENESIS_URL="https://raw.githubusercontent.com/pokt-network/pocket-network-genesis/master/shannon/${NETWORK}/genesis.json"
    curl -s -o $HOME/.poktroll/config/genesis.json "$GENESIS_URL"
    
    # Extract required version from genesis file
    POKTROLLD_VERSION=$(jq -r '.app_version' < $HOME/.poktroll/config/genesis.json)
    echo "Required poktrolld version for testnet-alpha: $POKTROLLD_VERSION"
    ```
    
    > **Warning:** Testnet Alpha is an unstable testing network. It may be reset frequently and is used for early testing of new features.
  </TabItem>
  
  <TabItem value="mainnet" label="Mainnet">
    ```bash
    # Set network to mainnet (production network)
    NETWORK="mainnet"
    
    # Create config directory if it doesn't exist
    mkdir -p $HOME/.poktroll/config
    
    # Download genesis file
    GENESIS_URL="https://raw.githubusercontent.com/pokt-network/pocket-network-genesis/master/shannon/${NETWORK}/genesis.json"
    curl -s -o $HOME/.poktroll/config/genesis.json "$GENESIS_URL"
    
    # Extract required version from genesis file
    POKTROLLD_VERSION=$(jq -r '.app_version' < $HOME/.poktroll/config/genesis.json)
    echo "Required poktrolld version for mainnet: $POKTROLLD_VERSION"
    ```
    
    > **Note:** Mainnet is the production network. Make sure you understand the requirements and responsibilities before joining mainnet.
  </TabItem>
</Tabs>

### 6. Install `poktrolld`

Now that we have the required version information from the genesis file, we can install the correct version of `poktrolld`:

```bash
# Determine your OS type and architecture
OS_TYPE=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
if [ "$ARCH" = "x86_64" ]; then
    ARCH="amd64"
elif [ "$ARCH" = "aarch64" ]; then
    ARCH="arm64"
fi

# Use the version extracted from the genesis file in the previous step
# POKTROLLD_VERSION was already set in the previous step

# Download and install poktrolld
RELEASE_URL="https://github.com/pokt-network/poktroll/releases/download/v${POKTROLLD_VERSION}/poktroll_${OS_TYPE}_${ARCH}.tar.gz"
mkdir -p $HOME/.poktroll/cosmovisor/genesis/bin $HOME/.local/bin
curl -L "$RELEASE_URL" | tar -zxvf - -C $HOME/.poktroll/cosmovisor/genesis/bin
chmod +x $HOME/.poktroll/cosmovisor/genesis/bin/poktrolld
ln -sf $HOME/.poktroll/cosmovisor/genesis/bin/poktrolld $HOME/.local/bin/poktrolld
```

### 7. Network Configuration

Initialize your node and configure it to connect to the network:

<Tabs groupId="network">
  <TabItem value="testnet-beta" label="Testnet Beta" default>
    ```bash
    # Extract chain-id from existing genesis
    CHAIN_ID=$(jq -r '.chain_id' < $HOME/.poktroll/config/genesis.json)
    
    # Initialize the node with your chosen moniker (node name)
    poktrolld init "YourNodeMoniker" --chain-id="$CHAIN_ID" --home=$HOME/.poktroll
    
    # Get seeds from the official repository
    SEEDS_URL="https://raw.githubusercontent.com/pokt-network/pocket-network-genesis/master/shannon/testnet-beta/seeds"
    SEEDS=$(curl -s "$SEEDS_URL")
    sed -i -e "s|^seeds *=.*|seeds = \"$SEEDS\"|" $HOME/.poktroll/config/config.toml
    
    # Configure external address for P2P communication
    EXTERNAL_IP=$(curl -s https://api.ipify.org)
    sed -i -e "s|^external_address *=.*|external_address = \"${EXTERNAL_IP}:26656\"|" $HOME/.poktroll/config/config.toml
    ```
  </TabItem>
  
  <TabItem value="testnet-alpha" label="Testnet Alpha">
    ```bash
    # Extract chain-id from existing genesis
    CHAIN_ID=$(jq -r '.chain_id' < $HOME/.poktroll/config/genesis.json)
    
    # Initialize the node with your chosen moniker (node name)
    poktrolld init "YourNodeMoniker" --chain-id="$CHAIN_ID" --home=$HOME/.poktroll
    
    # Get seeds from the official repository
    SEEDS_URL="https://raw.githubusercontent.com/pokt-network/pocket-network-genesis/master/shannon/testnet-alpha/seeds"
    SEEDS=$(curl -s "$SEEDS_URL")
    sed -i -e "s|^seeds *=.*|seeds = \"$SEEDS\"|" $HOME/.poktroll/config/config.toml
    
    # Configure external address for P2P communication
    EXTERNAL_IP=$(curl -s https://api.ipify.org)
    sed -i -e "s|^external_address *=.*|external_address = \"${EXTERNAL_IP}:26656\"|" $HOME/.poktroll/config/config.toml
    ```
  </TabItem>
  
  <TabItem value="mainnet" label="Mainnet">
    ```bash
    # Extract chain-id from existing genesis
    CHAIN_ID=$(jq -r '.chain_id' < $HOME/.poktroll/config/genesis.json)
    
    # Initialize the node with your chosen moniker (node name)
    poktrolld init "YourNodeMoniker" --chain-id="$CHAIN_ID" --home=$HOME/.poktroll
    
    # Get seeds from the official repository
    SEEDS_URL="https://raw.githubusercontent.com/pokt-network/pocket-network-genesis/master/shannon/mainnet/seeds"
    SEEDS=$(curl -s "$SEEDS_URL")
    sed -i -e "s|^seeds *=.*|seeds = \"$SEEDS\"|" $HOME/.poktroll/config/config.toml
    
    # Configure external address for P2P communication
    EXTERNAL_IP=$(curl -s https://api.ipify.org)
    sed -i -e "s|^external_address *=.*|external_address = \"${EXTERNAL_IP}:26656\"|" $HOME/.poktroll/config/config.toml
    ```
  </TabItem>
</Tabs>

### 8. Sync Options: Genesis vs Snapshot

You have two options to synchronize your node with the network:

#### Option 1: Sync from Genesis

Syncing from genesis validates the entire blockchain history but takes significantly longer.

If you're using this method, skip to the next section.

#### Option 2: Sync from Snapshot (Faster)

Using a snapshot allows you to quickly get your node operational by downloading a recent copy of the blockchain data.

<Tabs groupId="network">
  <TabItem value="testnet-beta" label="Testnet Beta" default>
    ```bash
    # Create a directory for the snapshot download
    SNAPSHOT_DIR="$HOME/poktroll_snapshot"
    mkdir -p "$SNAPSHOT_DIR"
    cd "$SNAPSHOT_DIR"
    
    # Base URL for snapshots
    SNAPSHOT_BASE_URL="https://snapshots.us-nj.poktroll.com"
    
    # Get latest snapshot information for testnet-beta
    LATEST_SNAPSHOT_HEIGHT=$(curl -s "$SNAPSHOT_BASE_URL/testnet-beta-latest-archival.txt")
    echo "Latest snapshot height: $LATEST_SNAPSHOT_HEIGHT"
    
    # Get snapshot version (important for compatibility)
    SNAPSHOT_VERSION=$(curl -s "$SNAPSHOT_BASE_URL/testnet-beta-${LATEST_SNAPSHOT_HEIGHT}-version.txt")
    echo "Snapshot version: $SNAPSHOT_VERSION"
    
    # If snapshot version is different from genesis version, you need to install that version instead
    if [ "$SNAPSHOT_VERSION" != "$POKTROLLD_VERSION" ]; then
        echo "Snapshot version ($SNAPSHOT_VERSION) differs from genesis version ($POKTROLLD_VERSION)"
        echo "Need to install the snapshot version for compatibility"
        
        # Update the POKTROLLD_VERSION and reinstall
        POKTROLLD_VERSION=$SNAPSHOT_VERSION
        RELEASE_URL="https://github.com/pokt-network/poktroll/releases/download/v${POKTROLLD_VERSION}/poktroll_${OS_TYPE}_${ARCH}.tar.gz"
        
        mkdir -p $HOME/.poktroll/cosmovisor/genesis/bin
        curl -L "$RELEASE_URL" | tar -zxvf - -C $HOME/.poktroll/cosmovisor/genesis/bin
        chmod +x $HOME/.poktroll/cosmovisor/genesis/bin/poktrolld
        ln -sf $HOME/.poktroll/cosmovisor/genesis/bin/poktrolld $HOME/.local/bin/poktrolld
    fi
    
    # Make sure your installed poktrolld matches the required version
    poktrolld version
    echo "Installed version must match: $POKTROLLD_VERSION"
    
    # Download via torrent (recommended method)
    TORRENT_URL="${SNAPSHOT_BASE_URL}/testnet-beta-latest-archival.torrent"
    aria2c --seed-time=0 --file-allocation=none --continue=true \
           --max-connection-per-server=4 --max-concurrent-downloads=16 --split=16 \
           --bt-enable-lpd=true --bt-max-peers=100 --bt-prioritize-piece=head,tail \
           --bt-seed-unverified \
           "$TORRENT_URL"
    
    # Find the downloaded file
    DOWNLOADED_FILE=$(find . -type f -name "*.tar.*" | head -n 1)
    
    # Extract the snapshot
    if [[ "$DOWNLOADED_FILE" == *.tar.zst ]]; then
        echo "Extracting .tar.zst snapshot..."
        zstd -d "$DOWNLOADED_FILE" --stdout | tar -xf - -C $HOME/.poktroll/data
    elif [[ "$DOWNLOADED_FILE" == *.tar.gz ]]; then
        echo "Extracting .tar.gz snapshot..."
        tar -zxf "$DOWNLOADED_FILE" -C $HOME/.poktroll/data
    else
        echo "Unknown snapshot format: $DOWNLOADED_FILE"
        exit 1
    fi
    
    # Clean up after extraction
    cd $HOME
    rm -rf "$SNAPSHOT_DIR"
    ```
  </TabItem>
  
  <TabItem value="testnet-alpha" label="Testnet Alpha">
    ```bash
    # Create a directory for the snapshot download
    SNAPSHOT_DIR="$HOME/poktroll_snapshot"
    mkdir -p "$SNAPSHOT_DIR"
    cd "$SNAPSHOT_DIR"
    
    # Base URL for snapshots
    SNAPSHOT_BASE_URL="https://snapshots.us-nj.poktroll.com"
    
    # Get latest snapshot information for testnet-alpha
    LATEST_SNAPSHOT_HEIGHT=$(curl -s "$SNAPSHOT_BASE_URL/testnet-alpha-latest-archival.txt")
    echo "Latest snapshot height: $LATEST_SNAPSHOT_HEIGHT"
    
    # Get snapshot version (important for compatibility)
    SNAPSHOT_VERSION=$(curl -s "$SNAPSHOT_BASE_URL/testnet-alpha-${LATEST_SNAPSHOT_HEIGHT}-version.txt")
    echo "Snapshot version: $SNAPSHOT_VERSION"
    
    # If snapshot version is different from genesis version, you need to install that version instead
    if [ "$SNAPSHOT_VERSION" != "$POKTROLLD_VERSION" ]; then
        echo "Snapshot version ($SNAPSHOT_VERSION) differs from genesis version ($POKTROLLD_VERSION)"
        echo "Need to install the snapshot version for compatibility"
        
        # Update the POKTROLLD_VERSION and reinstall
        POKTROLLD_VERSION=$SNAPSHOT_VERSION
        RELEASE_URL="https://github.com/pokt-network/poktroll/releases/download/v${POKTROLLD_VERSION}/poktroll_${OS_TYPE}_${ARCH}.tar.gz"
        
        mkdir -p $HOME/.poktroll/cosmovisor/genesis/bin
        curl -L "$RELEASE_URL" | tar -zxvf - -C $HOME/.poktroll/cosmovisor/genesis/bin
        chmod +x $HOME/.poktroll/cosmovisor/genesis/bin/poktrolld
        ln -sf $HOME/.poktroll/cosmovisor/genesis/bin/poktrolld $HOME/.local/bin/poktrolld
    fi
    
    # Make sure your installed poktrolld matches the required version
    poktrolld version
    echo "Installed version must match: $POKTROLLD_VERSION"
    
    # Download via torrent (recommended method)
    TORRENT_URL="${SNAPSHOT_BASE_URL}/testnet-alpha-latest-archival.torrent"
    aria2c --seed-time=0 --file-allocation=none --continue=true \
           --max-connection-per-server=4 --max-concurrent-downloads=16 --split=16 \
           --bt-enable-lpd=true --bt-max-peers=100 --bt-prioritize-piece=head,tail \
           --bt-seed-unverified \
           "$TORRENT_URL"
    
    # Find the downloaded file
    DOWNLOADED_FILE=$(find . -type f -name "*.tar.*" | head -n 1)
    
    # Extract the snapshot
    if [[ "$DOWNLOADED_FILE" == *.tar.zst ]]; then
        echo "Extracting .tar.zst snapshot..."
        zstd -d "$DOWNLOADED_FILE" --stdout | tar -xf - -C $HOME/.poktroll/data
    elif [[ "$DOWNLOADED_FILE" == *.tar.gz ]]; then
        echo "Extracting .tar.gz snapshot..."
        tar -zxf "$DOWNLOADED_FILE" -C $HOME/.poktroll/data
    else
        echo "Unknown snapshot format: $DOWNLOADED_FILE"
        exit 1
    fi
    
    # Clean up after extraction
    cd $HOME
    rm -rf "$SNAPSHOT_DIR"
    ```
  </TabItem>
  
  <TabItem value="mainnet" label="Mainnet">
    ```bash
    # Create a directory for the snapshot download
    SNAPSHOT_DIR="$HOME/poktroll_snapshot"
    mkdir -p "$SNAPSHOT_DIR"
    cd "$SNAPSHOT_DIR"
    
    # Base URL for snapshots
    SNAPSHOT_BASE_URL="https://snapshots.us-nj.poktroll.com"
    
    # Get latest snapshot information for mainnet
    LATEST_SNAPSHOT_HEIGHT=$(curl -s "$SNAPSHOT_BASE_URL/mainnet-latest-archival.txt")
    echo "Latest snapshot height: $LATEST_SNAPSHOT_HEIGHT"
    
    # Get snapshot version (important for compatibility)
    SNAPSHOT_VERSION=$(curl -s "$SNAPSHOT_BASE_URL/mainnet-${LATEST_SNAPSHOT_HEIGHT}-version.txt")
    echo "Snapshot version: $SNAPSHOT_VERSION"
    
    # If snapshot version is different from genesis version, you need to install that version instead
    if [ "$SNAPSHOT_VERSION" != "$POKTROLLD_VERSION" ]; then
        echo "Snapshot version ($SNAPSHOT_VERSION) differs from genesis version ($POKTROLLD_VERSION)"
        echo "Need to install the snapshot version for compatibility"
        
        # Update the POKTROLLD_VERSION and reinstall
        POKTROLLD_VERSION=$SNAPSHOT_VERSION
        RELEASE_URL="https://github.com/pokt-network/poktroll/releases/download/v${POKTROLLD_VERSION}/poktroll_${OS_TYPE}_${ARCH}.tar.gz"
        
        mkdir -p $HOME/.poktroll/cosmovisor/genesis/bin
        curl -L "$RELEASE_URL" | tar -zxvf - -C $HOME/.poktroll/cosmovisor/genesis/bin
        chmod +x $HOME/.poktroll/cosmovisor/genesis/bin/poktrolld
        ln -sf $HOME/.poktroll/cosmovisor/genesis/bin/poktrolld $HOME/.local/bin/poktrolld
    fi
    
    # Make sure your installed poktrolld matches the required version
    poktrolld version
    echo "Installed version must match: $POKTROLLD_VERSION"
    
    # Download via torrent (recommended method)
    TORRENT_URL="${SNAPSHOT_BASE_URL}/mainnet-latest-archival.torrent"
    aria2c --seed-time=0 --file-allocation=none --continue=true \
           --max-connection-per-server=4 --max-concurrent-downloads=16 --split=16 \
           --bt-enable-lpd=true --bt-max-peers=100 --bt-prioritize-piece=head,tail \
           --bt-seed-unverified \
           "$TORRENT_URL"
    
    # Find the downloaded file
    DOWNLOADED_FILE=$(find . -type f -name "*.tar.*" | head -n 1)
    
    # Extract the snapshot
    if [[ "$DOWNLOADED_FILE" == *.tar.zst ]]; then
        echo "Extracting .tar.zst snapshot..."
        zstd -d "$DOWNLOADED_FILE" --stdout | tar -xf - -C $HOME/.poktroll/data
    elif [[ "$DOWNLOADED_FILE" == *.tar.gz ]]; then
        echo "Extracting .tar.gz snapshot..."
        tar -zxf "$DOWNLOADED_FILE" -C $HOME/.poktroll/data
    else
        echo "Unknown snapshot format: $DOWNLOADED_FILE"
        exit 1
    fi
    
    # Clean up after extraction
    cd $HOME
    rm -rf "$SNAPSHOT_DIR"
    ```
  </TabItem>
</Tabs>

### 9. Set Up `systemd` Service

Create a systemd service to manage your node. You can customize the service name if you plan to run multiple nodes:

```bash
# Set a service name (change if running multiple nodes)
SERVICE_NAME="cosmovisor-poktroll"  # or another name like "cosmovisor-testnet"

sudo tee /etc/systemd/system/${SERVICE_NAME}.service > /dev/null <<EOF
[Unit]
Description=Cosmovisor daemon for poktrolld
After=network-online.target

[Service]
User=poktroll
ExecStart=/home/poktroll/.local/bin/cosmovisor run start --home=/home/poktroll/.poktroll
Restart=always
RestartSec=3
LimitNOFILE=infinity
LimitNPROC=infinity
Environment="DAEMON_NAME=poktrolld"
Environment="DAEMON_HOME=/home/poktroll/.poktroll"
Environment="DAEMON_RESTART_AFTER_UPGRADE=true"
Environment="DAEMON_ALLOW_DOWNLOAD_BINARIES=true"
Environment="UNSAFE_SKIP_BACKUP=true"

[Install]
WantedBy=multi-user.target
EOF
```

Enable and start the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable ${SERVICE_NAME}.service
sudo systemctl start ${SERVICE_NAME}.service
```

### 10. Configure your Firewall

To ensure your node can properly participate in the P2P network, you need to make port `26656` accessible from the internet. This is essential for communication with other nodes.

Choose the appropriate method for your system:

1. **Using UFW**:

   ```bash
   sudo ufw allow 26656/tcp
   ```

2. **Using iptables**:

   ```bash
   sudo iptables -A INPUT -p tcp --dport 26656 -j ACCEPT
   ```

3. **Cloud Provider Settings**: If running on a cloud provider (AWS, GCP, Azure, etc.), configure security groups or firewall rules to allow inbound traffic on port 26656.

4. **Verify your port** is accessible:

   ```bash
   # Install netcat if not already installed
   sudo apt install -y netcat
   
   # Use an external service to check port accessibility
   nc -zv portquiz.net 26656
   
   # Or have someone from outside check your port
   # nc -zv YOUR_EXTERNAL_IP 26656
   ```

### 11. Monitor Your Node

Check the status of your node:

```bash
# View service status
sudo systemctl status ${SERVICE_NAME}

# View logs in real-time
sudo journalctl -u ${SERVICE_NAME} -f

# Check sync status
poktrolld status | jq '.SyncInfo'
```

Your node is fully synced when `catching_up` is `false`.

You have now successfully set up a Full Node on the Pocket Network! This node can be used as a foundation to set up a validator, supplier, or gateway in the future.
