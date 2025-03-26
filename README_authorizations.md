- Pull `plan-mainnet-genesis` branch. Changes are captured in https://github.com/pokt-network/poktroll/pull/1152

  - Generate genesis

    ```bash
    # Where to save the files to
    export MAINNET_DIR=/Users/dk/pocket/tmp/mainnet

    # Make sure the dir is created
    mkdir -p $MAINNET_DIR

    # Probably good idea to run on a version/commit we want to launch mainnet on, and run a `go_develop` at this point

    # Generate the most important file ❤️ along with other important files (first validator key, configs..)
    ignite chain init --skip-proto --check-dependencies --clear-cache --home=$MAINNET_DIR
    ```

  - Add authz authorizations:

    ```bash
    # Set PNF and GROVE account addresses (should be known at this point)

    export PNF_ADDRESS=pokt123
    export GROVE_ADDRESS=pokt456

    sed -i'' -e "s/ADD_PNF_ADDRESS_HERE/$PNF_ADDRESS/g" pnf_authorizations.json
    sed -i'' -e "s/ADD_GROVE_ADDRESS_HERE/$GROVE_ADDRESS/g" grove_authorizations.json

    jq --argjson authz "$(cat pnf_authorizations.json)" \
       '.app_state.authz.authorization = $authz' \
       "$MAINNET_DIR/config/genesis.json" > tmp.json

    jq --argjson authz "$(cat grove_authorizations.json)" \
       '.app_state.authz.authorization += $authz' \
       tmp.json > tmp2.json

    mv tmp2.json "$MAINNET_DIR/config/genesis.json"
    rm tmp.json

    ```

  - Update dao_reward_address

    ```bash
    # use PNF_ADDRESS from previous snippet

    jq --arg addr "$PNF_ADDRESS" \
       '.app_state.tokenomics.params.dao_reward_address = $addr' \
       "$MAINNET_DIR/config/genesis.json" > tmp.json \
       && mv tmp.json "$MAINNET_DIR/config/genesis.json"
    ```

- TODOs:
  - Go through `TODO_` in the `config.yml` - there are some left from `TODO_BETA` that might apply.
