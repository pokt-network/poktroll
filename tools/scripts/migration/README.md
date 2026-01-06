# Morse -> Shannon Migration

## Morse to Shannon MainNet Migration

The state was migrated at height `96281`.

The transaction hash is `E4E74523CAC90E021AD181DBD1C6569AB0FD53FFBA2C4C639C963918F80869B3`.

It can be verified like so:

```bash
pocketd query \
    --node=https://sauron-rpc.infra.pocket.network tx \
    --type=hash E4E74523CAC90E021AD181DBD1C6569AB0FD53FFBA2C4C639C963918F80869B3 \
    >> /tmp/E4E74523CAC90E021AD181DBD1C6569AB0FD53FFBA2C4C639C963918F80869B3.txt
```

## Migration Artifacts

| Shannon Network | Morse MainNet Snapshot                                                                                                      | Height | Morse MainNet State Export                                                               | Morse TestNet Snapshot                                                                                                                                          | Height | Morse TestNet State Export                                                                       | `MsgImportMorseClaimableAccounts`                                                                      |
| --------------- | --------------------------------------------------------------------------------------------------------------------------- | ------ | ---------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------- |--------|--------------------------------------------------------------------------------------------------|--------------------------------------------------------------------------------------------------------|
| Alpha TestNet   | pruned-165398-165498-2025-04-15.tar                                                                                         | 165497 | [morse_account_state_alpha.json](./morse_account_state_alpha.json)                       | N/A                                                                                                                                                             | N/A    | N/A                                                                                              | [msg_import_morse_accounts_alpha.json](./msg_import_morse_accounts_alpha.json)                         |
| Beta TestNet    | [pruned-170406-170617-2025-06-03.tar](http://23.83.185.137/pruned-170406-170617-2025-06-03.tar) | 170616 | [morse_state_export_170616_2025-06-03.json](./morse_state_export_170616_2025-06-03.json) | [morse-testnet-179148-2025-06-01.tar](https://link.storjshare.io/raw/jwuhrvaepamwmqaywx6y57ygxdha/pocket-network-snapshots/morse-testnet-179148-2025-06-01.tar) | 179148 | [morse_testnet_state_export_179148_2025-06-01.json](./morse_state_export_179148_2025-06-01.json) | [msg_import_morse_accounts_m170616_t179148.json](./msg_import_morse_accounts_m170616_t179148.json)     |
| MainNet         | [pruned-170406-170617-2025-06-03.tar](http://23.83.185.137/pruned-170406-170617-2025-06-03.tar)                             | 170616 | [morse_state_export_170616_2025-06-03.json](./morse_state_export_170616_2025-06-03.json) | N/A                                                                                                                                                             | N/A    | N/A                                                                                              | [msg_import_morse_accounts_170616_2025-06-03.json](./msg_import_morse_accounts_170616_2025-06-03.json) |
