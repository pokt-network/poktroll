#!/bin/bash

set -e

NUM_APPS=10
APP_FUND_COIN=10000000000upokt

function wait_for_tx() {
  TX_HASH=$1

  sleep 2

  TX_LOG=$(make query_tx_log HASH=$TX_HASH | tr -d '"')
  if [[ -n $TX_LOG ]]; then
    echo ${TX_LOG}
    exit 1
  fi
}

#echo "Deleting existing application keys matching pattern 'app-*'..."
#rm -rf ~/.poktroll/keyring-test/app-*
#
#echo "Creating application ${NUM_APPS}..."
#
#APP_ADDRESSES=()
#APP_KEY_NAMES=()
#for i in $(seq 1 $NUM_APPS); do
#  APP_KEY_NAME=app-$((i + 100))
#  APP_KEY_NAMES+=(${APP_KEY_NAME})
#  poktrolld keys add ${APP_KEY_NAME} --keyring-backend test > /dev/null 2>&1
#  APP_ADDRESSES+=($(poktrolld keys show -a ${APP_KEY_NAME}))
#done
#
#echo "APP_ADDRESSES: ${APP_ADDRESSES[@]}"
#
#echo "Applications created, funding accounts..."
#FAUCET_ADDR=$(poktrolld keys show -a pnf --keyring-backend test)
#TX_HASH=$(poktrolld tx bank multi-send ${FAUCET_ADDR} $(echo ${APP_ADDRESSES[@]}) ${APP_FUND_COIN} --gas auto --keyring-backend test --from pnf --yes -o json | jq -r .txhash)
#wait_for_tx $TX_HASH
#
#echo "Accounts funded, staking & delegating to appgateserver1..."
#
#for APP_KEY_NAME in ${APP_KEY_NAMES[@]}; do
#  TX_HASH=$(poktrolld tx application stake-application --config localnet/poktrolld/config/application1_stake_config.yaml --from ${APP_KEY_NAME} --yes --keyring-backend test -o json | jq -r .txhash)
#  wait_for_tx $TX_HASH
#
#  TX_HASH=$(poktrolld tx application delegate-to-gateway pokt15vzxjqklzjtlz7lahe8z2dfe9nm5vxwwmscne4 --from ${APP_KEY_NAME} --yes --keyring-backend test -o json | jq -r .txhash)
#  wait_for_tx $TX_HASH
#done

echo "Applications ready, loading sessions..."

APP_ADDRESSES=$(poktrolld query application list-application --output json | jq -r '.applications[].address')

for APP_ADDRESS in $APP_ADDRESSES; do
#for APP_ADDRESS in ${APP_ADDRESSES[@]}; do
  echo "Loading application ${APP_ADDRESS}"
  hey -n 1000 -c 10 -m POST -H "Content-Type: application/json" -d '{"method":"eth_blockNumber","params":[],"id":1,"jsonrpc":"2.0"}' http://localhost:42079/anvil?applicationAddr=${APP_ADDRESS} > /dev/null 2>&1
done