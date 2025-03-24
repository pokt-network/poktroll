#!/bin/bash

cat <<EOF
This script runs the full node using cosmovisor and performs an upgrade after the upgrade plan is submitted on chain.
It simulates a real network upgrade. For consensus-breaking changes, ensure the 'old' binary doesn't have these changes.

Pre-requisites:
1. 'Old' binary
2. 'New' binary
3. Cosmovisor (Install: 'go install cosmossdk.io/tools/cosmovisor/cmd/cosmovisor@v1.5.0')
   Documentation: https://docs.cosmos.network/main/build/tooling/cosmovisor
4. \`upgrade.Upgrade\` with matching \`POCKETD_UPGRADE_PLAN_NAME\` created and included in the new version
5. LocalNet turned off
EOF

# Define paths and upgrade name
POCKETD_OLD_BINARY_PATH=$HOME/pocket/pokrtoll-for-releases/pocketd
POCKETD_NEW_BINARY_PATH=$HOME/pocket/pocket/pocketd
POCKETD_UPGRADE_PLAN_NAME=v0.0.4

cat <<EOF

The script will use the following:
Old binary: $POCKETD_OLD_BINARY_PATH
New binary: $POCKETD_NEW_BINARY_PATH
Upgrade plan name: $POCKETD_UPGRADE_PLAN_NAME

EOF

read -p "Do you want to continue? (y/n): " answer
if [[ $answer != "y" ]]; then
    echo "Script execution cancelled."
    exit 0
fi

# Cosmovisor settings:
export DAEMON_ALLOW_DOWNLOAD_BINARIES=false
export DAEMON_RESTART_AFTER_UPGRADE=true
export DAEMON_NAME=pocketd
export DAEMON_HOME=$HOME/.pocket # `localnet_regenesis` creates new genesis in this directory by default.

# Cleans up old upgrade binary and home dir.
rm -rf $DAEMON_HOME

# Runs regenesis.
make localnet_regenesis

# Setups cosmovisor directories and pocket binaries. On real network cosmovisor can download the binaries using onchain
# data when `DAEMON_ALLOW_DOWNLOAD_BINARIES=true`.
mkdir -p $DAEMON_HOME/cosmovisor/genesis/bin/ $DAEMON_HOME/cosmovisor/upgrades/$POCKETD_UPGRADE_PLAN_NAME/bin/
cp -r $POCKETD_OLD_BINARY_PATH $DAEMON_HOME/cosmovisor/genesis/bin/pocketd
cp -r $POCKETD_NEW_BINARY_PATH $DAEMON_HOME/cosmovisor/upgrades/$POCKETD_UPGRADE_PLAN_NAME/bin/pocketd

cosmovisor run start --home $DAEMON_HOME
