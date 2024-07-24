#!/bin/bash  

# The script starts node locally (no LocalNet) with cosmovisor and waits for an upgrade.
# Cosmovisor: https://docs.cosmos.network/main/build/tooling/cosmovisor
# Install with `go install cosmossdk.io/tools/cosmovisor/cmd/cosmovisor@v1.5.0`

export DAEMON_NAME=poktrolld 
# Pre-build binaries
# export POKTROLLD_OLD_BINARY_PATH=$HOME/pocket/poktroll/bin/poktrolld-v1
# export POKTROLLD_V2_PATH=$HOME/pocket/poktroll/bin/poktrolld-v2
export POKTROLLD_HOME=$HOME/.poktroll

# Cosmovisor directory.
export TMP_COSMOVISOR_DIR=/Users/dk/pocket/poktroll/bin/cosmovisor

# update if needed 
export DAEMON_ALLOW_DOWNLOAD_BINARIES=false 
export DAEMON_RESTART_AFTER_UPGRADE=true

# setup cored binaries 
# cp -r $POKTROLLD_OLD_BINARY_PATH $TMP_COSMOVISOR_DIR/genesis/bin/poktrolld
# cp -r $POKTROLLD_V2_PATH $TMP_COSMOVISOR_DIR/upgrades/v2/bin/poktrolld

# cp -R $TMP_COSMOVISOR_DIR $POKTROLLD_HOME/
cosmovisor run start --home $POKTROLLD_HOME # --minimum-gas-prices=0.00001upokt
