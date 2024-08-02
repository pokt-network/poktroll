#!/bin/bash
# This script submits a transaction to bump the block size to 6mb on LocalNet.
# Can be used as an example to submit other types of param changes.
# TODO_DOCUMENT(@okdas): Document how to write JSON and submit transactions for other parameter changes.

echo "Current params:"
poktrolld query consensus params

echo "Submitting transaction to change the block size..."
poktrolld tx authz exec tools/scripts/params/consensus_block_size_6mb.json --from pnf --yes

# Waiting for 3 seconds to query the new consensus params after the transaction has been processed by the validator.
# We have quick block creation times on LocalNet, so 3 seconds will suffice.
# TODO_CONSIDERATION: switch to `poktrolld query wait-tx`
echo "Waiting for 3 seconds..."
sleep 3

echo "New params:"
poktrolld query consensus params
