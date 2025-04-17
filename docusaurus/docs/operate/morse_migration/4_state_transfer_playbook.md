---
title: State Transfer Playbook
sidebar_position: 4
---

This page is intended for the Foundation (Authority) or whoever is managing the state transfer process.

## Table of Contents <!-- omit in toc -->

- [1. Retrieve a Pruned Morse Snapshot](#1-retrieve-a-pruned-morse-snapshot)
- [2. Export Morse State](#2-export-morse-state)
- [2. Transform Export to Canonical Account State](#2-transform-export-to-canonical-account-state)
- [3. Distribute Account State](#3-distribute-account-state)
- [4. Align on Social Consensus](#4-align-on-social-consensus)
- [5. Import Canonical State into Shannon](#5-import-canonical-state-into-shannon)
- [State Validation: Morse Account Holders](#state-validation-morse-account-holders)
  - [Why Validate?](#why-validate)
  - [How to Validate (Copy-pasta)](#how-to-validate-copy-pasta)

---

### 1. Retrieve a Pruned Morse Snapshot

Go to [Liquify's Snapshot Explorer](https://pocket-snapshot-uk.liquify.com/#/pruned/) and download the latest pruned snapshot.

Export the snapshot into a new directory on your local machine.

```bash
mkdir -p $HOME/morse-snapshot
tar -xvf <snapshot-file>.tar.gz -C $HOME/morse-snapshot
```

**⚠️ Note the height and date of the snapshot⚠️**

### 2. Export Morse State

Choose the snapshot height, which must be less than or equal to the snapshot height retrieved above. **This will be the published canonical export height.**

```bash
pocket util export-genesis-for-reset <HEIGHT> pocket --datadir $HOME/morse-snapshot > morse_state_export.json
```

:::tip Testing Workaround (for core developers only)

If you don't have a real snapshot (yet), you can generate a test `morse_state_export.json` file via the
following end-to-end test:

```bash
# make localnet_up # expectation that you are familiar with this process
make test_e2e_migration_fixture
mv e2e/tests/morse_state_export.json morse_state_export.json
```

:::

### 2. Transform Export to Canonical Account State

```bash
pocketd tx migration collect-morse-accounts morse_state_export.json morse_account_state.json
```

:::warning Troubleshooting

Ensure your `--home` flag points to a Shannon directory if you see this error:

```bash
failed to read in /Users/olshansky/.pocket/config/config.toml: While parsing config: toml: invalid character at start of key: {
```

:::

### 3. Distribute Account State

Distribute the `morse_account_state.json` and its hash for public verification by Morse account/stake-holders.

### 4. Align on Social Consensus

- Wait for consensus (offchain, time-bounded).
- React to feedback as needed.

### 5. Import Canonical State into Shannon

```bash
pocketd tx migration import-morse-accounts morse_account_state.json --from <authorized-key-name> --grpc-addr=<shannon-network-grpc-endpoint>
```

For example, to run the above on different networks:

```bash
# LocalNet
pocketd tx migration import-morse-accounts morse_account_state.json --from <authorized-key-name> --grpc-addr=https://shannon-grove-grpc.alpha.poktroll.com

# Alpha TestNet
pocketd tx migration import-morse-accounts morse_account_state.json --from <authorized-key-name> --grpc-addr=https://shannon-grove-grpc.alpha.poktroll.com

# Beta TestNet
pocketd tx migration import-morse-accounts morse_account_state.json --from <authorized-key-name> --grpc-addr=https://shannon-grove-grpc.beta.poktroll.com

# MainNet
pocketd tx migration import-morse-accounts morse_account_state.json --from <authorized-key-name> --grpc-addr=https://shannon-grove-grpc.mainnet.poktroll.com
```

### State Validation: Morse Account Holders

:::info Fun Analogy 👯

- It's like making sure you and your friends (your accounts) are on "the list" before it gets printed out and handed to the crypto-club bouncer.
- Double-check that all names are on the list and spelled correctly; **the bouncer at crypto-club is brutally strict**.
  :::

#### Why Validate?

- The ETVL process determines the _official_ set of claimable Morse accounts (balances/stakes) on Shannon.
- It's **critical** for Morse account/stake-holders to confirm their account(s) are included and correct in the proposed `MsgImportMorseClaimableAccounts`.

#### How to Validate (Copy-pasta)

- **Download** the latest proposed `MsgImportMorseClaimableAccounts` (contains both the state and its hash):
  :::warning TODO_MAINNET
  Link to latest published [`MsgImportMorseClaimableAccounts`](https://github.com/pokt-network/poktroll/blob/main/proto/pocket/migration/tx.proto#L50) proposal.
  :::

- **Validate** using the Shannon CLI:
  - Example file: `./msg_import_morse_claimable_accounts.json`
  - Run:
    ```bash
    # TODO_MAINNET_CRITICAL(@bryanchriswhite, #1034): Complete this example once the CLI is available.
    pocketd tx migration validate-morse-accounts ./msg_import_morse_claimable_accounts.json [morse_hex_address1, ...]
    ```
  - You can pass multiple Morse addresses to the command.
  - For each address, the corresponding `MorseClaimableAccount` is printed for manual inspection and validation.
