#!/bin/bash

echo "Current params:"
poktrolld query consensus params

echo "Submitting transaction to change the block size..."
poktrolld tx authz exec tools/scripts/params/consensus_increase_block_size.json --from pnf --yes

sleep 3
echo "New params:"
poktrolld query consensus params


# DISCUSS_IN_THIS_PR: do we need to add authorization to change staking params? Consider other modules as well.
# ❯ poktrolld query staking params
# params:
#   bond_denom: upokt
#   historical_entries: 10000
#   max_entries: 7
#   max_validators: 1
#   min_commission_rate: "0"
#   unbonding_time: 504h0m0s