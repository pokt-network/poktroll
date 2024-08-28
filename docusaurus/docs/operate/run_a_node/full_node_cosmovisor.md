---
title: Full Node - Cosmovisor
sidebar_position: 1
---

# Full Node - Cosmovisor <!-- omit in toc -->

- [What is a Full Node](#what-is-a-full-node)
- [What is Cosmovisor](#what-is-cosmovisor)
- [Download and set up Cosmovisor](#download-and-set-up-cosmovisor)
- [Setting up Cosmovisor](#setting-up-cosmovisor)
- [Upgrading with Cosmovisor](#upgrading-with-cosmovisor)

## What is a Full Node

In blockchain networks, a full node retains continuous synchs and updates the latest copy of the
ledger. It may either be pruned full node (the latest data only) or an archival full node (including
complete and historical data).

You can visit the [Cosmos SDK documentation](https://docs.cosmos.network/main/user/run-node/run-node)
for more information on Full Nodes.

## What is Cosmovisor

As an alternative to our [Full Node - Docker](./full_node_docker.md) guide, we also provide documentation on how to deploy
a Full Node using Cosmovisor.

[Cosmovisor](https://docs.cosmos.network/main/build/tooling/cosmovisor) is a tool that automates the version management
for our blockchain. It allows operators to automatically upgrade their full nodes and validators without downtime and
reduce maintenance overhead.

:::info
TODO(@okdas): finish this tutorial as a part of [#526](https://github.com/pokt-network/poktroll/issues/526).
:::

## Download and set up Cosmovisor

0. Prerequisites

If you're logged in as `root` user, let's create a separate user:

```bash
# ONLY NEEDED IF YOU LOGGED IN UNDER ROOT

# Add user
useradd -m -s /bin/bash poktroll-cosmovisor

# Set 
passwd poktroll-cosmovisor

# Add the new user to the sudo group
usermod -aG sudo poktroll-cosmovisor

# Switch to the new user
su - poktroll-cosmovisor
```

1. Download Cosmovisor:

```bash
# Determine architecture and download the appropriate Cosmovisor binary
ARCH=$(uname -m)
if [ "$ARCH" = "x86_64" ]; then 
  ARCH="amd64"
elif [ "$ARCH" = "aarch64" ]; then 
  ARCH="arm64"
fi
curl -LO "https://github.com/cosmos/cosmos-sdk/releases/download/cosmovisor%2Fv1.6.0/cosmovisor-v1.6.0-linux-${ARCH}.tar.gz"

# Create $HOME/bin if it doesn't exist and extract the binary
mkdir -p $HOME/bin
tar -zxvf cosmovisor-v1.6.0-linux-${ARCH}.tar.gz -C $HOME/bin cosmovisor

# Ensure $HOME/bin is in the PATH
if ! grep -q 'export PATH=$HOME/bin:$PATH' $HOME/.profile; then
  echo 'export PATH=$HOME/bin:$PATH' >> $HOME/.profile
fi

# Source the updated .profile to apply changes to the current session
source $HOME/.profile

# Clean up
rm cosmovisor-v1.6.0-linux-${ARCH}.tar.gz
```

2. Set up Cosmovisor:

```bash
echo "export DAEMON_NAME=poktrolld" >> ~/.profile
echo "export DAEMON_HOME=$HOME/.poktroll" >> ~/.profile
echo "export DAEMON_RESTART_AFTER_UPGRADE=true" >> ~/.profile
echo "export DAEMON_ALLOW_DOWNLOAD_BINARIES=true" >> ~/.profile
echo "export UNSAFE_SKIP_BACKUP=true" >> ~/.profile
source ~/.profile
```

3. Set up systemd unit

<!-- TODO: EXPAND HERE. ADD SYSTEMD UNIT, MAKE IT STARTABLE ON RESTART AND START SERVICE -->
```bash
echo "[Unit]
Description=Cosmovisor daemon for poktrolld
After=network-online.target

[Service]
Environment=\"DAEMON_NAME=${DAEMON_NAME}\"
Environment=\"DAEMON_HOME=${DAEMON_HOME}\"
Environment=\"DAEMON_RESTART_AFTER_UPGRADE=${DAEMON_RESTART_AFTER_UPGRADE}\"
Environment=\"DAEMON_ALLOW_DOWNLOAD_BINARIES=${DAEMON_ALLOW_DOWNLOAD_BINARIES}\"
Environment=\"UNSAFE_SKIP_BACKUP=${UNSAFE_SKIP_BACKUP}\"
ExecStart=${HOME}/go/bin/cosmovisor start --home=${DAEMON_HOME}
Restart=always
RestartSec=3
LimitNOFILE=infinity
LimitNPROC=infinity

[Install]
WantedBy=default.target
"
```

## Setting up Cosmovisor

[Content to be added]

## Upgrading with Cosmovisor

[Content to be added]