#!/usr/bin/env bash
# Re-generate genesis.json file

GENESIS_PATH="./localnet/genesis.json"
POCKETD_HOME="./localnet/pocketd"
BEGINNING_OF_TIMES="2023-04-20T20:04:20.069420Z"

# If no `jq`, then error and exit
if ! command -v jq &>/dev/null; then
    echo "jq is required but not installed. Please install jq and try again."
    exit 1
fi

# Save existing values that will change after regenesis,
# so we can restore them later (and avoid unnecessary changes in git history)
GENTX0_PUBKEY=$(jq -r '.app_state.genutil.gen_txs[0].body.messages[0].pubkey.key' $GENESIS_PATH)
GENTX0_MEMO=$(jq -r '.app_state.genutil.gen_txs[0].body.memo' $GENESIS_PATH)
GENTX0_SIGNATURE=$(jq -r '.app_state.genutil.gen_txs[0].signatures[0]' $GENESIS_PATH)

# Run regenesis
# NOTE: intentionally not using --home <dir> flag to avoid overwriting the test keyring
faketime "$BEGINNING_OF_TIMES" /bin/bash -c 'ignite chain init --skip-proto'

# only copy new files, without changin the existing keys
# cp -rn ${HOME}/.pocket/keyring-test ${POCKETD_HOME}
cp -r ${HOME}/.pocket/keyring-test ${POCKETD_HOME}

# copy the new genesis into the git repo
cp ${HOME}/.pocket/config/genesis.json ./localnet/

# Now, let's put old values back and set the "correct" date
# jq --arg pubkey "$GENTX0_PUBKEY" \
#     --arg signature "$GENTX0_SIGNATURE" \
#     --arg memo "$GENTX0_MEMO" \
#     '.app_state.genutil.gen_txs[0].body.messages[0].pubkey.key=$pubkey |
#  .app_state.genutil.gen_txs[0].body.memo=$memo |
#  .app_state.genutil.gen_txs[0].signatures[0]=$signature' $GENESIS_PATH > tmp.json && mv tmp.json $GENESIS_PATH
