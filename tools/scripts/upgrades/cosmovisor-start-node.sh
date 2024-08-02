#!/bin/bash

cat << EOF
This script runs the full node using cosmovisor and performs an upgrade after the upgrade plan is submitted on chain. 
It simulates a real network upgrade. For consensus-breaking changes, ensure the 'old' binary doesn't have these changes.

Pre-requisites:
1. 'Old' binary
2. 'New' binary
3. Cosmovisor (Install: 'go install cosmossdk.io/tools/cosmovisor/cmd/cosmovisor@v1.5.0')
   Documentation: https://docs.cosmos.network/main/build/tooling/cosmovisor
4. \`upgrade.Upgrade\` with matching \`POKTROLLD_UPGRADE_PLAN_NAME\` created and included in the new version
5. LocalNet turned off
EOF

# Define paths and upgrade name
POKTROLLD_OLD_BINARY_PATH=$HOME/pocket/pokrtoll-for-releases/poktrolld
POKTROLLD_NEW_BINARY_PATH=$HOME/pocket/poktroll/poktrolld
POKTROLLD_UPGRADE_PLAN_NAME=v0.0.4

cat << EOF

The script will use the following:
Old binary: $POKTROLLD_OLD_BINARY_PATH
New binary: $POKTROLLD_NEW_BINARY_PATH
Upgrade plan name: $POKTROLLD_UPGRADE_PLAN_NAME

EOF

read -p "Do you want to continue? (y/n): " answer
if [[ $answer != "y" ]]; then
    echo "Script execution cancelled."
    exit 0
fi

# Cosmovisor settings:
export DAEMON_ALLOW_DOWNLOAD_BINARIES=false 
export DAEMON_RESTART_AFTER_UPGRADE=true
export DAEMON_NAME=poktrolld 
export DAEMON_HOME=$HOME/.poktroll # `localnet_regenesis` creates new genesis in this directory by default.

# Cleans up old upgrade binary and home dir.
rm -rf $DAEMON_HOME

# Runs regenesis.
make localnet_regenesis

# Setups cosmovisor directories and poktroll binaries. On real network cosmovisor can download the binaries using on-chain
# data when `DAEMON_ALLOW_DOWNLOAD_BINARIES=true`.
mkdir -p $DAEMON_HOME/cosmovisor/genesis/bin/ $DAEMON_HOME/cosmovisor/upgrades/$POKTROLLD_UPGRADE_PLAN_NAME/bin/
cp -r $POKTROLLD_OLD_BINARY_PATH $DAEMON_HOME/cosmovisor/genesis/bin/poktrolld
cp -r $POKTROLLD_NEW_BINARY_PATH $DAEMON_HOME/cosmovisor/upgrades/$POKTROLLD_UPGRADE_PLAN_NAME/bin/poktrolld

cosmovisor run start --home $DAEMON_HOME