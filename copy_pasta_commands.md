## Create a `pocket` account

```bash
pocket accounts
pocket accounts create
pocket accounts create --datadir ./pocket_test
pocket accounts list --datadir ./pocket_test
# Update temp_state_export.json
pocket accounts export 2e2624762bcfee4a44001543adddce0e4f4cc823
```

## Create a `pocketd` account

```bash
pocketd --keyring-backend="$POCKET_TEST_KEYRING_BACKEND" --home="$POCKET_HOME_PROD" "$@
```

```
pocketd tx migration import-morse-accounts \
  "$MSG_IMPORT_MORSE_ACCOUNTS_PATH" \
  --from <authorized-key-name> \
  --grpc-addr=<shannon-network-grpc-endpoint> \
  --home <shannon-home-directory> \
  --chain-id=<shannon-chain-id> \
  --gas=auto --gas-adjustment=1.5
```
