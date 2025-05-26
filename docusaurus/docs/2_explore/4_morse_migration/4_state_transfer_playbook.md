---
title: State Transfer Playbook
sidebar_position: 4
---

This page is intended for the Foundation (Authority) or whoever is managing the state transfer process.

## Table of Contents <!-- omit in toc -->

- [Step by Step Instructions for Protocol Maintainer](#step-by-step-instructions-for-protocol-maintainer)
  - [1. Retrieve a Pruned Morse Snapshot](#1-retrieve-a-pruned-morse-snapshot)
  - [2. Export Morse Snapshot State](#2-export-morse-snapshot-state)
  - [3. Transform Morse Export to a Canonical Account State Import Message](#3-transform-morse-export-to-a-canonical-account-state-import-message)
  - [4. Distribute Canonical Account State Import Message](#4-distribute-canonical-account-state-import-message)
  - [5. Align on Account State via Social Consensus](#5-align-on-account-state-via-social-consensus)
  - [6. Import Canonical State into Shannon](#6-import-canonical-state-into-shannon)
  - [7. Query Canonical State in Shannon](#7-query-canonical-state-in-shannon)
  - [8. Cleanup \& Documentation](#8-cleanup--documentation)
- [State Validation: Morse Account Holders](#state-validation-morse-account-holders)
  - [Why Validate?](#why-validate)
  - [How to Validate](#how-to-validate)
- [Troubleshooting](#troubleshooting)
  - [I don't have a real snapshot on my machine](#i-dont-have-a-real-snapshot-on-my-machine)
  - [`invalid character at start of key`](#invalid-character-at-start-of-key)
  - [`failed to get grant with given granter: ..., grantee: ... & msgType: /pocket.migration.MsgImportMorseClaimableAccounts`](#failed-to-get-grant-with-given-granter--grantee---msgtype-pocketmigrationmsgimportmorseclaimableaccounts)
  - [`http2 frame too large`](#http2-frame-too-large)

---

## Step by Step Instructions for Protocol Maintainer

### 1. Retrieve a Pruned Morse Snapshot

Go to [Liquify's Snapshot Explorer](https://pocket-snapshot-uk.liquify.com/#/pruned/) and download the latest pruned snapshot.

:::warning

If you're reproducing the Morse state migration process, for validation purposes, you MUST use the **same snapshot height** in order to guarantee deterministic and correct results.

See the table in [Migration Artifacts](https://github.com/pokt-network/poktroll/tree/main/tools/scripts/migration) to
ensure the correct snapshot heights are used.

:::

Export the snapshot into a new directory on your local machine.

```bash
mkdir -p $HOME/morse-mainnet-snapshot
# 1. Untar the snapshot file
tar -xvf <snapshot-file>.tar -C $HOME/morse-mainnet-snapshot
# 2. Change directory to the extracted snapshot folder
cd $HOME/morse-mainnet-snapshot
```

:::warning Note the height and date of the snapshot

The height and date are encoded in the snapshot file name.

For example, the snapshot file name `pruned-166819-166919-2025-04-29.tar` has a max height of `166918` and a date of `2025-04-29`; the end-height is exclusive.

:::

### 2. Export Morse Snapshot State

Choose the snapshot height, which must be less than or equal to the snapshot height retrieved above. **This will be the published canonical export height.**

```bash
export MAINNET_SNAPSHOT_HEIGHT="<HEIGHT>" # E.g. "166918"
export MAINNET_SNAPSHOT_DATE="<DATE>" # E.g. "2025-04-29"
export MORSE_MAINNET_STATE_EXPORT_PATH="./morse_state_export_${MAINNET_SNAPSHOT_HEIGHT}_${MAINNET_SNAPSHOT_DATE}.json"
pocket --datadir="$HOME/morse-mainnet-snapshot" util export-genesis-for-reset "$MAINNET_SNAPSHOT_HEIGHT" pocket > "$MORSE_MAINNET_STATE_EXPORT_PATH"
```

### 3. Transform Morse Export to a Canonical Account State Import Message

:::info Testing on Shannon TestNet?

`MorseClaimableAccount`s imported to Shannon TestNet(s) are a merge of the Morse MainNet and TestNet state exports, for developer convenience.

See [Migration Testing](12_testnet_testing.md) for additional instructions.
Follow the steps there, and resume **from the next step (i.e. skip this step)**.

:::

```bash
export MSG_IMPORT_MORSE_ACCOUNTS_PATH="./msg_import_morse_accounts_${MAINNET_SNAPSHOT_HEIGHT}_${MAINNET_SNAPSHOT_DATE}.json"
pocketd tx migration collect-morse-accounts "$MORSE_MAINNET_STATE_EXPORT_PATH" "$MSG_IMPORT_MORSE_ACCOUNTS_PATH"
```

### 4. Distribute Canonical Account State Import Message

Distribute the `msg_import_morse_accounts_${SNAPSHOT_HEIGHT}_${SNAPSHOT_DATE}.json` and its hash for public verification by Morse account/stake-holders.

### 5. Align on Account State via Social Consensus

- Wait for consensus (offchain, time-bounded).
- React to feedback as needed.

### 6. Import Canonical State into Shannon

:::danger

This can **ONLY BE DONE ONCE** on networks with the `allow_morse_account_import_overwrite` param disabled (e.g. MainNet).

:::

The following `import-morse-accounts` command can be used to import the canonical account state into Shannon:

```bash
pocketd tx migration import-morse-accounts \
  "$MSG_IMPORT_MORSE_ACCOUNTS_PATH" \
  --from <authorized-key-name> \
  --home <shannon-home-directory> \
  --chain-id=<shannon-chain-id> \
  --gas=auto --gas-adjustment=1.5
```

<details>
<summary>Convenience functions for `import-morse-accounts` by network</summary>

```bash
# LocalNet
pocketd tx migration import-morse-accounts "$MSG_IMPORT_MORSE_ACCOUNTS_PATH" --from pnf --home=./localnet/pocketd --chain-id=pocket --gas=auto --gas-prices=1upokt --gas-adjustment=1.5

# Alpha TestNet
pocketd tx migration import-morse-accounts "$MSG_IMPORT_MORSE_ACCOUNTS_PATH" --from pokt1r6ja6rz6rpae58njfrsgs5n5sp3r36r2q9j04h --home=~/.pocket_prod --chain-id=pocket-alpha --gas=auto --gas-prices=1upokt --gas-adjustment=1.5 --node=http://shannon-testnet-grove-rpc.alpha.poktroll.com

# Beta TestNet
pocketd tx migration import-morse-accounts "$MSG_IMPORT_MORSE_ACCOUNTS_PATH" --from pokt1f0c9y7mahf2ya8tymy8g4rr75ezh3pkklu4c3e --home=~/.pocket_prod --chain-id=pocket-beta --gas=auto --gas-prices=1upokt --gas-adjustment=1.5 --node=http://shannon-testnet-grove-rpc.beta.poktroll.com

# MainNet
pocketd tx migration import-morse-accounts "$MSG_IMPORT_MORSE_ACCOUNTS_PATH" --from pokt18808wvw0h4t450t06uvauny8lvscsxjfyua7vh --home=~/.pocket_prod --chain-id=pocket-mainnet --gas=auto --gas-prices=1upokt --gas-adjustment=1.5 --node=http://shannon-grove-rpc.mainnet.poktroll.com
```

</details>

### 7. Query Canonical State in Shannon

The list of all claimable Morse accounts (balances/stakes) on Shannon can be retrieved using the following command:

```bash
pocketd query migration list-morse-claimable-account
```

A specific Morse address can be retrieved using the following command:

```bash
pocketd query migration show-morse-claimable-account <morse-address>
```

<details>
<summary>Convenience functions for `list-morse-claimable-account` by network</summary>

```bash
# LocalNet
pocketd query migration list-morse-claimable-account --node http://localhost:26657

# Alpha TestNet
pocketd query migration list-morse-claimable-account --node https://shannon-grove-rpc.alpha.poktroll.com

# Beta TestNet
pocketd query migration list-morse-claimable-account --node https://shannon-grove-rpc.beta.poktroll.com

# MainNet
pocketd query migration list-morse-claimable-account --node https://shannon-grove-rpc.mainnet.poktroll.com
```

</details>

### 8. Cleanup & Documentation

Document the details of the snapshot upload in [tools/scripts/migration/README.md](https://github.com/pokt-network/poktroll/blob/main/tools/scripts/migration/README.md)/.

## State Validation: Morse Account Holders

:::info Fun Analogy 👯

- It's like making sure you and your friends (your accounts) are on "the list" before it gets printed out and handed to the crypto-club bouncer.
- Double-check that all names are on the list and spelled correctly; **the bouncer at crypto-club is brutally strict**.

:::

### Why Validate?

**The output in `msg_morse_import_accounts_${SNAPSHOT_HEIGHT}_${DATE}.json` MUST be validated before step 6.**

The [State Transfer Overview](3_state_transfer_overview.md) process determines the _official_ set of claimable Morse accounts (balances/stakes) on Shannon.

It's **critical** for Morse account/stake-holders to confirm their account(s) are included and correct in the proposed `msg_import_morse_claimable_accounts.json`.

### How to Validate

Firstly, **Retrieve** the latest proposed `msg_morse_import_accounts_${SNAPSHOT_HEIGHT}_${DATE}.json` (contains both the state and its hash).

Then, **Validate** using the Shannon CLI like so:

```bash
pocketd tx migration validate-morse-accounts ./msg_import_morse_accounts_<SNAPSHOT_HEIGHT>_<SNAPSHOT_DATE>.json [morse_hex_address1, ...]
```

- You can pass multiple Morse addresses to the command
- For each address, the corresponding `MorseClaimableAccount` is printed for manual inspection and validation

## Troubleshooting

### I don't have a real snapshot on my machine

Intended for core developer who need a `morse_state_export.json` for testing.

Use the following E2E test:

```bash
# make localnet_up # expectation that you are familiar with this process
make test_e2e_migration_fixture
mv e2e/tests/morse_state_export.json morse_state_export.json
```

### `invalid character at start of key`

If you're getting this error, make sure your `--home` flag points to a Shannon (not Morese) directory:

```bash
failed to read in /Users/olshansky/.pocket/config/config.toml: While parsing config: toml: invalid character at start of key: {
```

### `failed to get grant with given granter: ..., grantee: ... & msgType: /pocket.migration.MsgImportMorseClaimableAccounts`

You can query all grants by a given granter (i.e. `pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t`) like so:

```bash
pocketd query authz grants-by-granter pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t
```

And if one is missing, simply execute it like so:

```bash
pocketd tx authz grant \
  pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw \
  generic \
  --msg-type="/pocket.migration.MsgImportMorseClaimableAccounts" \
  --from pnf_alpha \
  --expiration 16725225600 \
  --chain-id pocket-alpha \
  --gas auto --gas-prices 1upokt --gas-adjustment 1.5 \
  --node=http://localhost:26657 \
  --home=$HOME/.pocket_prod
```

### `http2 frame too large`

If you're seeing the following issue:

```bash
rpc error: code = Unavailable desc = connection error: desc = "error reading server preface: http2: frame too large"
```

The http/grpc configs of the `RPC_ENDPOINT` you're using may need to be configured.

If you're running it yourself in `k8s`, a workaround can be to replace this command:

```
pocketd tx migration import-morse-accounts ./tools/scripts/migration/morse_account_state_alpha.json  --from pnf_alpha  --home=$HOME/.pocket_prod --chain-id=pocket-alpha --gas=auto --gas-prices=1upokt --gas-adjustment=1.5 --node=https://shannon-testnet-grove-rpc.alpha.poktroll.com
```

with

```bash

kubectl port-forward pods/alpha-validator1-pocketd-0 26657:26657 9090:9090 -n testnet-alpha --address 0.0.0.0

pocketd tx migration import-morse-accounts ./tools/scripts/migration/morse_account_state_alpha.json  --from pnf_alpha  --home=$HOME/.pocket_prod --chain-id=pocket-alpha --gas=auto --gas-prices=1upokt --gas-adjustment=1.5 --node=localhost:26657
```
