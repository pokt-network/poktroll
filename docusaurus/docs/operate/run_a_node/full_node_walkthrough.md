---
title: Full Node (systemd)
sidebar_position: 2
---

## Run a Validator <!-- omit in toc -->

This walkthrough provides step-by-step instructions to manually install and configure a Full Node from scratch.

- [Introduction](#introduction)
- [Pre-Requisites](#pre-requisites)
- [Step 1: Create a New User](#step-1-create-a-new-user)
- [Step 2: Install Dependencies](#step-2-install-dependencies)
- [Step 3: Set Up Environment Variables](#step-3-set-up-environment-variables)
- [Step 4: Install Cosmovisor](#step-4-install-cosmovisor)
- [Step 5: Install `poktrolld`](#step-5-install-poktrolld)
- [Step 6: Configure `poktrolld`](#step-6-configure-poktrolld)
- [Step 7: Set Up `systemd` Service](#step-7-set-up-systemd-service)
- [Step 8: Open Firewall Ports](#step-8-open-firewall-ports)
- [Next Steps](#next-steps)

### Introduction

This guide will help you install a Full Node for Pocket Network manually, giving you control over each step of the process. Running a Full Node is the first step toward becoming a Validator.

**TL;DR**: If you're comfortable using an automated script, check out the [Full Node Cheat Sheet](../quickstart/full_node_cheatsheet.md) for quick setup instructions.

### Pre-Requisites

- **Linux-based System**: Preferably Debian-based distributions.
- **Root or Sudo Access**: Administrative privileges are required.
- **Dedicated Server or Virtual Machine**: Any provider is acceptable.

### Step 1: Create a New User

Create a dedicated user to run `poktrolld`:

```bash
sudo adduser poktroll
```

Set a password when prompted, and add the user to the sudo group:

```bash
sudo usermod -aG sudo poktroll
```

### Step 2: Install Dependencies

Update your package list and install necessary dependencies:

```bash
sudo apt-get update
sudo apt-get install -y curl tar wget jq
```

### Step 3: Set Up Environment Variables

Switch to the `poktroll` user and set environment variables required for Cosmovisor:

```bash
sudo su - poktroll
```

Add the following to your `.profile`:

```bash
echo "export DAEMON_NAME=poktrolld" >> ~/.profile
echo "export DAEMON_HOME=\$HOME/.poktroll" >> ~/.profile
echo "export DAEMON_RESTART_AFTER_UPGRADE=true" >> ~/.profile
echo "export DAEMON_ALLOW_DOWNLOAD_BINARIES=true" >> ~/.profile
echo "export UNSAFE_SKIP_BACKUP=false" >> ~/.profile
source ~/.profile
```

### Step 4: Install Cosmovisor

Download and install Cosmovisor:

:::info
Alternatively, you can follow the [official cosmovisor installation instructions](https://docs.cosmos.network/main/build/tooling/cosmovisor#installation).
:::

```bash
mkdir -p $HOME/bin
COSMOVISOR_VERSION="v1.6.0"
ARCH=$(uname -m)
if [ "$ARCH" = "x86_64" ]; then 
    ARCH="amd64"
elif [ "$ARCH" = "aarch64" ]; then 
    ARCH="arm64"
fi

curl -L "https://github.com/cosmos/cosmos-sdk/releases/download/cosmovisor%2F${COSMOVISOR_VERSION}/cosmovisor-${COSMOVISOR_VERSION}-linux-${ARCH}.tar.gz" | tar -zxvf - -C $HOME/bin
echo 'export PATH=$HOME/bin:$PATH' >> ~/.profile
source ~/.profile
```


### Step 5: Install `poktrolld`

Download and install `poktrolld`:

1. **Extract Version and Set Architecture**:

   Different networks start with different versions. Let's extract the version from the genesis file and set the architecture.

   ```bash
   # Extract version from genesis.json
   POKTROLLD_VERSION=$(jq -r '.app_version' < $HOME/.poktroll/config/genesis.json)
   ARCH=$(uname -m)
   if [ "$ARCH" = "x86_64" ]; then ARCH="amd64"
   elif [ "$ARCH" = "aarch64" ]; then ARCH="arm64"
   fi
   ```

2. **Download and Install the Binary**:

   Create the cosmovisor genesis directory and download the binary.
   ```bash
   mkdir -p $HOME/.poktroll/cosmovisor/genesis/bin
   curl -L "https://github.com/pokt-network/poktroll/releases/download/v${POKTROLLD_VERSION}/poktroll_linux_${ARCH}.tar.gz" | tar -zxvf - -C $HOME/.poktroll/cosmovisor/genesis/bin
   chmod +x $HOME/.poktroll/cosmovisor/genesis/bin/poktrolld
   ln -sf $HOME/.poktroll/cosmovisor/genesis/bin/poktrolld $HOME/bin/poktrolld
   ```

### Step 6: Configure `poktrolld`

Initialize configuration files and set up the node:

1. **Select Network and Download Genesis**:

   ```bash
   # Select network (testnet-alpha, testnet-beta, or mainnet)
   NETWORK="testnet-beta"  # Change this to your desired network
   
   # Download genesis file
   GENESIS_URL="https://raw.githubusercontent.com/pokt-network/pocket-network-genesis/master/shannon/${NETWORK}/genesis.json"
   curl -s -o $HOME/.poktroll/config/genesis.json "$GENESIS_URL"
   
   # Extract chain-id from genesis
   CHAIN_ID=$(jq -r '.chain_id' < $HOME/.poktroll/config/genesis.json)
   ```

2. **Initialize the Node**:

   ```bash
   poktrolld init "YourNodeMoniker" --chain-id="$CHAIN_ID" --home=$HOME/.poktroll
   ```

3. **Set Seeds**:

   ```bash
   SEEDS_URL="https://raw.githubusercontent.com/pokt-network/pocket-network-genesis/master/shannon/${NETWORK}/seeds"
   SEEDS=$(curl -s "$SEEDS_URL")
   sed -i -e "s|^seeds *=.*|seeds = \"$SEEDS\"|" $HOME/.poktroll/config/config.toml
   ```

4. **Set External Address**:

   ```bash
   EXTERNAL_IP=\$(curl -s https://api.ipify.org)
   sed -i -e "s|^external_address *=.*|external_address = \"\${EXTERNAL_IP}:26656\"|" \$HOME/.poktroll/config/config.toml
   ```

### Step 7: Set Up `systemd` Service

Create a `systemd` service file to manage the node:

```bash
sudo tee /etc/systemd/system/cosmovisor.service > /dev/null <<EOF
[Unit]
Description=Cosmovisor daemon for poktrolld
After=network-online.target

[Service]
User=poktroll
ExecStart=/home/poktroll/bin/cosmovisor run start --home=/home/poktroll/.poktroll
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
sudo systemctl enable cosmovisor.service
sudo systemctl start cosmovisor.service
```

### Step 8: Open Firewall Ports

To ensure your node can properly participate in the P2P network, you need to make port `26656` accessible from the internet. This may involve:

1. **Configuring your firewall**:
   
   For UFW:
   ```bash
   sudo ufw allow 26656/tcp
   ```

   For iptables:
   ```bash
   sudo iptables -A INPUT -p tcp --dport 26656 -j ACCEPT
   ```

2. **Cloud Provider Settings**: 
   - If running on a cloud provider (AWS, GCP, Azure, etc.), ensure you configure the security groups or firewall rules to allow inbound traffic on port 26656.
   
3. **Router Configuration**:
   - If running behind a router, configure port forwarding for port 26656 to your node's internal IP address.

You can verify your port is accessible using a tool like netcat or telnet from another machine:
```bash
nc -vz your_server_ip 26656
```

### Next Steps

Your Full Node is now up and running. You can check its status and logs using the commands:

- **Check Status**:

  ```bash
  sudo systemctl status cosmovisor.service
  ```

- **View Logs**:

  ```bash
  sudo journalctl -u cosmovisor.service -f
  ```

<!-- 
## Becoming a Validator

TODO(@okdas, #754): Add instructions for becoming a validator.

-->
