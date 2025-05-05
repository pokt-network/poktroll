#!/bin/bash

cat <<EOF
Cosmovisor Full Node Upgrade Script

- Runs the full node using cosmovisor
- Performs an upgrade after the upgrade plan is submitted on-chain
- Simulates a real network upgrade

IMPORTANT for consensus-breaking changes:
   - Ensure the 'old' binary does NOT include these changes

Prerequisites:
   - 'Old' binary
   - 'New' binary
   - Cosmovisor
       - Install: go install cosmossdk.io/tools/cosmovisor/cmd/cosmovisor@v1.6.0
       - Docs: https://docs.cosmos.network/main/build/tooling/cosmovisor
       - TODO_TECHDEBT(@olshansk): Upgrade up from v1.6.0 once we are confident a newer version is stable (v1.7.x is not)
   - upgrade.Upgrade with matching POCKETD_UPGRADE_PLAN_NAME created and included in the new version
   - LocalNet turned off
EOF

# Define paths and upgrade name
POCKETD_OLD_BINARY_PATH=$HOME/pocket/pocket-for-releases/pocketd
POCKETD_NEW_BINARY_PATH=$HOME/pocket/pocket/pocketd
# TODO_IN_THIS_PR: Should this be updated?
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
