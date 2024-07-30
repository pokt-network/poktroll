#!/bin/bash

# Submit upgrade
echo "Submitting an upgrade transaction via authz..."
poktrolld tx authz exec tools/scripts/upgrades/authz_upgrade_tx.json --from pnf --yes

echo "Sleeping for 3 seconds..."
sleep 3

echo "A scheduled upgrade plan:"
poktrolld query upgrade plan

# If changing consensus module parameters (such as block size), execute after an upgrade to verify the block size
# has been changed.
# 
# params:
#   abci: {}
#   block:
#     max_bytes: "22020096"
# 
# Should be increased to:
# 
# params:
#   abci: {}
#   block:
#     max_bytes: "44040192"
#     max_gas: "-1"

# poktrolld query consensus params