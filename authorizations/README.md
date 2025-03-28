# POKTROLL Mainnet Genesis Setup <!-- omit in toc -->

## Table of Contents <!-- omit in toc -->

- [Identifying the necessary authorizations](#identifying-the-necessary-authorizations)
  - [Cosmos SDK Messages](#cosmos-sdk-messages)
  - [Pocket Network Messages](#pocket-network-messages)
  - [Update the authorizations](#update-the-authorizations)
- [Genesis Preparation](#genesis-preparation)
  - [Generate Genesis](#generate-genesis)
- [Generation `authz` authorizations](#generation-authz-authorizations)
  - [Update DAO Reward Address](#update-dao-reward-address)

## Identifying the necessary authorizations

Use the following commands **to** see all the transactions available in the `Cosmos SDK` and `Pocket Network` modules.

Identify the functions you need and update [pnf_authorizations.json](./pnf_authorizations.json) and [grove_authorizations.json](./grove_authorizations.json) with the necessary authorizations.

The above requires a deep understanding of the protocol **which is** out of scope for this document.

### Cosmos SDK Messages

Identify all the messages available in the `Cosmos SDK` modules like so:

```bash
git clone https://github.com/cosmos/cosmos-sdk.git
cd cosmos-sdk
grep -r "message Msg" --include="*.proto" . | grep -v "Response" | sed -E 's/.*\/cosmos\/([^\/]+)\/[^:]+:message (Msg[^{]+).*/cosmos.\1.\2/' | sed 's/ {//' | grep "^cosmos\." | sort
```

Which will output

```bash
cosmos.accounts.MsgAuthenticate
cosmos.accounts.MsgCreateProposal
cosmos.accounts.MsgDelegate
cosmos.accounts.MsgExecute
cosmos.accounts.MsgExecuteBundle
...
```

### Pocket Network Messages

Identify all the messages available in the `Pocket Network` modules like so:

```bash
git clone git@github.com:pokt-network/poktroll.git
cd poktroll
grep -r "message Msg" --include="*.proto" . | grep -v "Response" | sed -E 's/.*\/poktroll\/([^\/]+)\/tx\.proto:message (Msg[^{]+).*/poktroll.\1.\2/' | sed 's/ {//' | grep "^poktroll\." | sort
```

Which will output

```bash
poktroll.application.MsgDelegateToGateway
poktroll.application.MsgStakeApplication
poktroll.application.MsgTransferApplication
poktroll.application.MsgUndelegateFromGateway
poktroll.application.MsgUnstakeApplication
poktroll.application.MsgUpdateParam
poktroll.application.MsgUpdateParams
...
```

### Update the authorizations

For each authorization you need, make sure to include a JSON object with the following structure:

```json
  {
    "granter": "pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t",
    "grantee": "poktREPLACE_THIS_WITH_THE_ADDRESS_THAT_SHOULD_GET_PERMISSION",
    "authorization": {
      "@type": "\/cosmos.authz.v1beta1.GenericAuthorization",
      "msg": "\/poktroll.service.MsgUpdateParams"
    },
    "expiration": "2500-01-01T00:00:00Z"
  },
```

The above object contains the following fields:

- `granter`: The onchain `x/gov` module address (`pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t`)
- `grantee`: The address that should get permission
- `authorization`: Enables authorization for `poktroll.service.MsgUpdateParams`
- `expiration`: Expires in the year `2500`

:::tip Recommended authorizations

You can pass the lists from Pocket and Cosmos to an LLM of your choice and get a few suggestions

:::

## Genesis Preparation

### Generate Genesis

```bash
# Where to save the files to
export MAINNET_DIR=$HOME/pocket/tmp/mainnet

# Make sure the dir is created
mkdir -p $MAINNET_DIR

# Make a note of the git sha you're on

# Generate the genesis file and other important configuration files (first validator key, configs..)
ignite chain init --skip-proto --check-dependencies --clear-cache --home=$MAINNET_DIR
```

## Generation `authz` authorizations

TODO:
- Update: ignite chain init --skip-proto --check-dependencies --clear-cache --config=config_mainnet.yml --home=$MAINNET_DIR
- Skip the export
- tar -czvf mainnet_backup.tar.gz $MAINNET_DIR

```bash
export PNF_ADDRESS=pokt_PNF_ADDRESS
export GROVE_ADDRESS=pokt_GROVE_ADDRESS

sed -i'' -e "s/ADD_PNF_ADDRESS_HERE/$PNF_ADDRESS/g" authorizations/pnf_authorizations.json
sed -i'' -e "s/ADD_GROVE_ADDRESS_HERE/$GROVE_ADDRESS/g" authorizations/grove_authorizations.json

jq --argjson authz "$(cat authorizations/pnf_authorizations.json)" \
   '.app_state.authz.authorization = $authz' \
   "$MAINNET_DIR/config/genesis.json" > tmp.json

jq --argjson authz "$(cat authorizations/grove_authorizations.json)" \
   '.app_state.authz.authorization += $authz' \
   tmp.json > tmp2.json

mv tmp2.json "$MAINNET_DIR/config/genesis.json"
`rm tmp.json`
```

### Update DAO Reward Address

```bash
# use PNF_ADDRESS from previous snippet
jq --arg addr "$PNF_ADDRESS" \
   '.app_state.tokenomics.params.dao_reward_address = $addr' \
   "$MAINNET_DIR/config/genesis.json" > tmp.json && \
   mv tmp.json "$MAINNET_DIR/config/genesis.json"
```
