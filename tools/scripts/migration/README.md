# Morse -> Shannon Migration

## Morse to Shannon MainNet Migration

The state was migrated at height `96281`.

The transaction hash is `E4E74523CAC90E021AD181DBD1C6569AB0FD53FFBA2C4C639C963918F80869B3`.

It can be verified like so:

```bash
pocketd query \
    --node=https://shannon-grove-rpc.mainnet.poktroll.com tx \
    --type=hash E4E74523CAC90E021AD181DBD1C6569AB0FD53FFBA2C4C639C963918F80869B3 \
    >> /tmp/E4E74523CAC90E021AD181DBD1C6569AB0FD53FFBA2C4C639C963918F80869B3.txt
```

## Migration Artifacts

| Shannon Network | Morse MainNet Snapshot                                                                                                      | Height | Morse MainNet State Export                                                               | Morse TestNet Snapshot                                                                                                                                          | Height | Morse TestNet State Export                                                                       | `MsgImportMorseClaimableAccounts`                                                                       |
| --------------- | --------------------------------------------------------------------------------------------------------------------------- | ------ | ---------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------- |
| Alpha TestNet   | pruned-165398-165498-2025-04-15.tar                                                                                         | 165497 | [morse_account_state_alpha.json](./morse_account_state_alpha.json)                       | N/A                                                                                                                                                             | N/A    | N/A                                                                                              | [msg_import_morse_accounts_alpha.json](./msg_import_morse_accounts_alpha.json)                          |
| Beta TestNet    | [pruned-169726-169826-2025-05-27.tar](https://pocket-snapshot.liquify.com/files/pruned/pruned-169726-169826-2025-05-27.tar) | 169825 | [morse_state_export_169825_2025-05-27.json](./morse_state_export_169825_2025-05-27.json) | [morse-testnet-179148-2025-06-01.tar](https://link.storjshare.io/raw/jwuhrvaepamwmqaywx6y57ygxdha/pocket-network-snapshots/morse-testnet-179148-2025-06-01.tar) | 176966 | [morse_testnet_state_export_176966_2025-05-09.json](./morse_state_export_176966_2025-05-09.json) | [msg_import_morse_accounts_m167639_t176966_beta.json](./msg_import_morse_accounts_m167639_t176966.json) |
| MainNet         | [pruned-170406-170617-2025-06-03.tar](http://23.83.185.137/pruned-170406-170617-2025-06-03.tar)                             | 170616 | [morse_state_export_170616_2025-06-03.json](./morse_state_export_170617_2025-06-03.json) | N/A                                                                                                                                                             | N/A    | N/A                                                                                              | [msg_import_morse_accounts_170617_2025-06-03.json](./msg_import_morse_accounts_170617_2025-06-03.json)  |
