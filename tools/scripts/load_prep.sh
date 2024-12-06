#!/bin/bash

set -e

function wait_sec() {
    echo "waiting $1 seconds for next block..."
    sleep $1
}

# Run in a separate terminal
# make localnet_down; make localnet_up

make acc_initialize_pubkeys
rm -rf ~/.poktroll/keyring-test/app-*

# update application_min_stake to 1upokt
make params_update_application_min_stake
wait_sec 3
make params_get_application

# update proof_request_probability to 1
make params_update_proof_proof_request_probability
wait_sec 3
make params_get_proof
