#!/bin/bash

description="This script is intended to run the full node using cosmovisor, and perform an upgrade after the upgrade plan is \
submitted on chain. It is helpful to simulate a real network upgrade. If the upgrade includes a consensus-breaking \
change, make sure the 'old' binary does not have these changes (so the real upgrade can be simulated)."

echo "$description"

echo -e "\nPre-requisites:"
echo "1. 'Old' binary."
echo "2. 'New' binary."
echo "3. Cosmovisor. Install with 'go install cosmossdk.io/tools/cosmovisor/cmd/cosmovisor@v1.5.0'"
echo "   Documentation: https://docs.cosmos.network/main/build/tooling/cosmovisor"
echo "4. The \`upgrade.Upgrade\` with matching \`POKTROLLD_UPGRADE_PLAN_NAME\` should be created and included in the new version."
echo "5. The LocalNet should be turned off."

echo -e "\nMake sure to adjust POKTROLLD_OLD_BINARY_PATH, POKTROLLD_NEW_BINARY_PATH and POKTROLLD_UPGRADE_PLAN_NAME in the script."
echo "Binaries can be built using 'ignite chain build' command and moved to a directory not included in git."

read -p "Do you want to continue? (y/n): " answer
if [[ $answer != "y" ]]; then
    echo "Script execution cancelled."
    exit 0
fi

# Paths to pre-built binaries.
export POKTROLLD_OLD_BINARY_PATH=$HOME/pocket/pokrtoll-for-releases/poktrolld
export POKTROLLD_NEW_BINARY_PATH=$HOME/pocket/poktroll/poktrolld
export POKTROLLD_UPGRADE_PLAN_NAME=v0.0.4

export DAEMON_HOME=$HOME/.poktroll # `localnet_regenesis` creates new genesis in this directory by default.

# Cosmovisor settings:
export DAEMON_ALLOW_DOWNLOAD_BINARIES=false 
export DAEMON_RESTART_AFTER_UPGRADE=true
export DAEMON_NAME=poktrolld 

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