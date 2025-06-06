---
title: Migration E2E Testing (TestNet)
sidebar_position: 12
---

## Table Of Contents <!-- omit in toc -->

- [Background](#background)
- [Step by Step Instructions for Protocol Maintainer](#step-by-step-instructions-for-protocol-maintainer)
  - [1. Retrieve a Morse TestNet Snapshot](#1-retrieve-a-morse-testnet-snapshot)
  - [2. Export Morse TestNet State](#2-export-morse-testnet-state)
  - [3. Export Morse MainNet State](#3-export-morse-mainnet-state)
  - [4. Merge Morse MainNet \& TestNet States](#4-merge-morse-mainnet--testnet-states)
  - [5. Complete the State Transform Process](#5-complete-the-state-transform-process)

---

## Background

For end-to-end migration testing (on a _live/public_ network with _real stakeholders_),
Morse State uploads on Shannon Beta TestNets:need to:

- Merge both the **Morse MainNet** and **TestNet state** exports into a single file.
- Maximizes developer convenience and peace of mind.

:::info Callout

On **Shannon TestNet only** will the snapshot contain both Morse MainNet and TestNet state.

:::

---

## Step by Step Instructions for Protocol Maintainer

**Background**:

- Morse state exports are derived from snapshots.
- TestNet snapshots are taken manually and distributed via STORJ.

### 1. Retrieve a Morse TestNet Snapshot

:::important Snapshot Height Verification

- Snapshot heights MUST match those used to generate the canonical state.

See the table in [Migration Artifacts](https://github.com/pokt-network/poktroll/tree/main/tools/scripts/migration) to ensure the correct snapshot heights and for all links.

:::

**Extract the snapshot by cop-pasting the following commands**:

```bash
mkdir -p $HOME/morse-testnet-snapshot
# Untar the snapshot file (replace with your downloaded filename)
tar -xvf <testnet-snapshot-file>.tar -C $HOME/morse-testnet-snapshot
cd $HOME/morse-testnet-snapshot
```

### 2. Export Morse TestNet State

**Set the snapshot height and date** (replace with actual values):

```bash
export TESTNET_SNAPSHOT_HEIGHT="176966"
export TESTNET_SNAPSHOT_DATE="2025-05-09"
export MORSE_TESTNET_STATE_EXPORT_PATH="./morse_state_export_${TESTNET_SNAPSHOT_HEIGHT}_${TESTNET_SNAPSHOT_DATE}.json"
```

**Export the state** (update `--datadir` if your snapshot path is different):

```bash
pocket --datadir="$HOME/morse-testnet-snapshot" util export-genesis-for-reset "$TESTNET_SNAPSHOT_HEIGHT" pocket > "$MORSE_TESTNET_STATE_EXPORT_PATH"
```

### 3. Export Morse MainNet State

Follow steps 1-2 in [State Transfer Playbook](./4_state_transfer_playbook.md) to retrieve and export the Morse MainNet snapshot.

### 4. Merge Morse MainNet & TestNet States

- Merge the Morse MainNet and TestNet state exports into a single file:

```bash
export MSG_IMPORT_MORSE_ACCOUNTS_PATH="./msg_import_morse_accounts_m${MAINNET_SNAPSHOT_HEIGHT}_t${TESTNET_SNAPSHOT_HEIGHT}.json"

pocketd tx migration collect-morse-accounts \
  "$MORSE_MAINNET_STATE_EXPORT_PATH" "$MSG_IMPORT_MORSE_ACCOUNTS_PATH" \
  --merge-state="$MORSE_TESTNET_STATE_EXPORT_PATH"
```

- **Replace** `${MAINNET_SNAPSHOT_HEIGHT}` and `${TESTNET_SNAPSHOT_HEIGHT}` with your actual values.
- **Result**: `msg_import_morse_accounts_m<MAINNET_SNAPSHOT_HEIGHT>_t<TESTNET_SNAPSHOT_HEIGHT>.json` is ready for import.

### 5. Complete the State Transform Process

Resume the [State Transfer Playbook](./4_state_transfer_playbook.md) from [step 3.1](./4_state_transfer_playbook.md#31-optional-on-shannon-testnet-and-mandatory-on-shannon-mainnet) to complete state transformation.
