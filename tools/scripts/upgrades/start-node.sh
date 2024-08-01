#!/bin/bash  

# The script starts a full node locally (not LocalNet) with cosmovisor and waits for an upgrade.
# Cosmovisor: https://docs.cosmos.network/main/build/tooling/cosmovisor
# Install with `go install cosmossdk.io/tools/cosmovisor/cmd/cosmovisor@v1.5.0`

export DAEMON_NAME=poktrolld 
# Pre-build binaries. Built with `ignite chain build --skip-proto --output .`
export POKTROLLD_OLD_BINARY_PATH=/Users/dk/pocket/pokrtoll-for-releases/poktrolld
export POKTROLLD_NEW_BINARY_PATH=/Users/dk/pocket/poktroll/poktrolld
export DAEMON_HOME=$HOME/.poktroll

# Cosmovisor directory.
# export TMP_COSMOVISOR_DIR=/Users/dk/pocket/poktroll/bin/cosmovisor

# update if needed 
export DAEMON_ALLOW_DOWNLOAD_BINARIES=false 
export DAEMON_RESTART_AFTER_UPGRADE=true

# Cleans up old upgrade binary and home dir
rm -rf $DAEMON_HOME

make localnet_regenesis

# Setup cosmovisor directories and provide an upgrade binary, as we test locally and don't want to pull the binary
# from the internet.
mkdir -p $DAEMON_HOME/cosmovisor/genesis/bin/ $DAEMON_HOME/cosmovisor/upgrades/v0.0.4/bin/
cp -r $POKTROLLD_OLD_BINARY_PATH $DAEMON_HOME/cosmovisor/genesis/bin/poktrolld
cp -r $POKTROLLD_NEW_BINARY_PATH $DAEMON_HOME/cosmovisor/upgrades/v0.0.4/bin/poktrolld

cosmovisor run start --home $DAEMON_HOME
