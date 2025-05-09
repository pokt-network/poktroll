---
title: Migration Testing
sidebar_position: 11
---

## Table Of Contents

- [Background](#background)
- [1. Retrieve a Morse TestNet Snapshot](#1-retrieve-a-morse-testnet-snapshot)
- [2. Export Morse TestNet Export State](#2-export-morse-testnet-export-state)
- [3. Transform & Merge Morse Exports to a Canonical Account State Import Message](#3-transform--merge-morse-exports-to-a-canonical-account-state-import-message)

## Background

For testing the migration process end-to-end, _on a live and public network and with real stakeholders_, the Morse state
is initially imported into Shannon TestNet(s).

In order to maximize both developer convenience and peace of mind, the Morse states which are imported into Shannon
TestNet(s) are **a merge of both a Morse MainNet and a Morse TestNet state exports**.

## 1. Retrieve a Morse TestNet Snapshot

Morse state exports are derived from snapshots. Since Morse TestNet snapshots are not automated, like Morse MainNet
snapshots are, Morse TestNet snapshots are taken manually and distributed via STORJ network.
Links are provided here for convenience:

| Snapshot                                                                                                                                                      | Height | Date       | Size   | 
|---------------------------------------------------------------------------------------------------------------------------------------------------------------|--------|------------|--------|
| [morse-tesnet-176681-2025-05-07.txz](https://link.storjshare.io/raw/jwndx6se4o6tdwpeqhxm7imiam6a/pocket-network-snapshots/morse-tesnet-176681-2025-05-07.txz) | 176681 | 2025-05-07 | 7.37GB |

In order to generate or reproduce the canonical merged Morse export state, the snapshot heights for used then
reproducing MUST match what those used when generating the canonical state.
See the table in [Migration Artifacts](https://github.com/pokt-network/poktroll/tree/main/tools/scripts/migration) to
ensure the correctness of snapshot heights.

Export the snapshot into a new directory on your local machine.

```bash
mkdir -p $HOME/morse-testnet-snapshot
# 1. Untar the snapshot file
tar -xvf <testnet-snapshot-file>.tar -C $HOME/morse-testnet-snapshot
# 2. Change directory to the extracted snapshot folder
cd $HOME/morse-testnet-snapshot
```

## 2. Export Morse TestNet Export State

Substitute the corresponding snapshot height and date to generate the Morse TestNet state export:

```bash
export TESTNET_SNAPSHOT_HEIGHT="<HEIGHT>" # E.g. "176681"
export TESTNET_SNAPSHOT_DATE="<DATE>" # E.g. "2025-05-07"
export MORSE_TESTNET_STATE_EXPORT_PATH="./morse_state_export_${TESTNET_SNAPSHOT_HEIGHT}_${TESTNET_SNAPSHOT_DATE}.json"
pocket --datadir="$HOME/morse-snapshot" util export-genesis-for-reset "$TESTNET_SNAPSHOT_HEIGHT" pocket > "$MORSE_TESTNET_STATE_EXPORT_PATH"
```

:::tip
Ensure that you've **also** retrieved and exported the corresponding Morse MainNet snapshot by following steps 1-2
in [State Transfer Playbook](./4_state_transfer_playbook.md).
:::

## 3. Transform & Merge Morse Exports to a Canonical Account State Import Message

Merge the Morse MainNet and TestNet state exports into a single
`msg_import_morse_acccounts_m<SNAPSHOT_HEIGHT>_t<TESTNET_SNAPSHOT_HEIGHT>.json` file:

```bash
export MSG_IMPORT_MORSE_ACCOUNTS_PATH="./msg_import_morse_accounts_m${SNAPSHOT_HEIGHT}_t${TESTNET_SNAPSHOT_HEIGHT}.json"
export MORSE_TESTNET_STATE_EXPORT_PATH="./morse_testnet_state_export${TESTNET_SNAPSHOT_HEIGHT}_${TESTNET_SNAPSHOT_DATE}.json"
pocketd tx migration collect-morse-accounts "$MORSE_STATE_EXPORT_PATH" "$MSG_IMPORT_MORSE_ACCOUNTS_PATH" \
  --merge-state="$MORSE_TESTNET_STATE_EXPORT_PATH"
```