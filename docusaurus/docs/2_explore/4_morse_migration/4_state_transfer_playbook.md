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
    - [3.1 Copy and verify the state import message:](#31-copy-and-verify-the-state-import-message)
    - [3.2 Optional on Shannon TestNet and mandatory on Shannon MainNet](#32-optional-on-shannon-testnet-and-mandatory-on-shannon-mainnet)
    - [3.3 Update migration artifacts in git](#33-update-migration-artifacts-in-git)
  - [4. Social Consensus](#4-social-consensus)
    - [4.1 Distribute Canonical Account State Import Message](#41-distribute-canonical-account-state-import-message)
    - [4.2 Align on Account State via Social Consensus](#42-align-on-account-state-via-social-consensus)
  - [5. Import Canonical State into Shannon](#5-import-canonical-state-into-shannon)
  - [6. Query Canonical State in Shannon](#6-query-canonical-state-in-shannon)
  - [7. Cleanup \& Documentation](#7-cleanup--documentation)
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

We're using `snapshot-pruned-165398-165498-2025-04-15.tar` as an example. Make sure to replace it with the latest snapshot file name.

### 1. Retrieve a Pruned Morse Snapshot

Go to [Liquify's Snapshot Explorer](https://pocket-snapshot-uk.liquify.com/#/pruned/) and download the latest pruned snapshot.

:::tip Multi-threaded Download

Use a tool like [aria2c](https://aria2.github.io/) for a faster multi-threaded download. **For example**:

```bash
aria2c -x 16 \
  -s 16 \
  -k 1M \
  https://pocket-snapshot-us.liquify.com/files/pruned/snapshot-pruned-165398-165498-2025-04-15.tar
```

:::

Export the snapshot into a new directory on your local machine.

For example, for the `snapshot-pruned-165398-165498-2025-04-15.tar` snapshot:

```bash
# Set snapshot variables (replace with actual values)
export SNAPSHOT_FILE="snapshot-pruned-165398-165498-2025-04-15.tar"
export SNAPSHOT_DIR="$HOME/morse-mainnet-snapshot"

# 1. Cleanup and create new directory
rm -rf "$SNAPSHOT_DIR"
mkdir -p "$SNAPSHOT_DIR"

# 2. Untar the snapshot file
tar -xvf ~/Downloads/"$SNAPSHOT_FILE" -C "$SNAPSHOT_DIR"
```

:::warning Note the height and date of the snapshot file name

The snapshot file name `snapshot-pruned-165398-165498-2025-04-15.tar` has a max height of `165497` and a date of `2025-04-15`. **The end-height is exclusive.**

:::

### 2. Export Morse Snapshot State

Choose the snapshot height, which must be less than or equal to the snapshot height retrieved above. **This will be the published canonical export height.**

Prepare your environment variables:

```bash
export MAINNET_SNAPSHOT_HEIGHT="165497"
export MAINNET_SNAPSHOT_DATE="2025-04-15"
export MORSE_MAINNET_STATE_EXPORT_PATH="./morse_state_export_${MAINNET_SNAPSHOT_HEIGHT}_${MAINNET_SNAPSHOT_DATE}.json"
```

Verify the snapshot data directory has the expected files:

```bash
ls "$SNAPSHOT_DIR"/data
# Expected output: application.db blockstore.db evidence.db state.db txindexer.db
```

Export the state:

```bash
pocket --datadir="$SNAPSHOT_DIR" util export-genesis-for-reset "$MAINNET_SNAPSHOT_HEIGHT" pocket > "$MORSE_MAINNET_STATE_EXPORT_PATH"
```

Verify the state export:

```bash
# Check the file was created
jq -r '{root_keys: .|keys, num_auth_accounts: .app_state.auth.accounts|length, num_nodes: .app_state.pos.validators|length, num_applications: .app_state.application.applications|length}' "$MORSE_MAINNET_STATE_EXPORT_PATH"

# Example output:
{
  "root_keys": [
    "app_hash",
    "app_state",
    "chain_id",
    "consensus_params",
    "genesis_time"
  ],
  "num_auth_accounts": 57630,
  "num_nodes": 12161,
  "num_applications": 2298
}
```

### 3. Transform Morse Export to a Canonical Account State Import Message

Prepare your environment variables:

```bash
export MSG_IMPORT_MORSE_ACCOUNTS_PATH="./msg_import_morse_accounts_${MAINNET_SNAPSHOT_HEIGHT}_${MAINNET_SNAPSHOT_DATE}.json"
```

Transform the Morse export to a canonical account state import message:

```bash
pocketd tx migration collect-morse-accounts "$MORSE_MAINNET_STATE_EXPORT_PATH" "$MSG_IMPORT_MORSE_ACCOUNTS_PATH"
```

:::note Shannon TestNet Only

`MorseClaimableAccount`s imported to Shannon TestNet(s) are a merge of the Morse MainNet and TestNet state exports, for developer convenience.

Follow **[the steps in "Migration E2E Testing"](./12_testnet_testing.md) instead of running the command above**, and then resume from step 3.1.

:::

#### 3.1 Copy and verify the state import message:

```bash
# Copy files to migration directory
cp "$MORSE_MAINNET_STATE_EXPORT_PATH" ./tools/scripts/migration/
cp "$MSG_IMPORT_MORSE_ACCOUNTS_PATH" ./tools/scripts/migration/
```

```bash
# Check the file was created, and that it contains the expected number of accounts and hash
jq -r '{num_morse_claimable_accounts: .morse_account_state.accounts|length, morse_account_state_hash: .morse_account_state_hash}' "$MSG_IMPORT_MORSE_ACCOUNTS_PATH"

# Example output:
{
  "num_morse_claimable_accounts": 63802,
  "morse_account_state_hash": "w5/Sf4c1L9/G5eYPC0wvrI3ynzwlnxcka0C8ULB9sMc="
}

```

#### 3.2 Optional on Shannon TestNet and mandatory on Shannon MainNet

Manually unstake all Morse validators on Shannon MainNet.

Use the [official list](https://docs.google.com/spreadsheets/d/1V33oAE01s7JLXxjsnJYt7cg_ttaEKbXFFs_qpLKR9l0/edit)
of validators that requested an auto-unstake to unstake them.

**Validator Groups to Unstake:**

- **QSpider**: `c409a9e0d1be8780fe0b29dcdf72f8a879fb110c,...`
- **Weavers** and **Spacebelt**: (See complete lists in validator spreadsheet)

**Automated Unstaking Process:**

```bash
# Use the prepared unstaking script
./tools/scripts/params/manual_unstake_prepared.sh "tools/scripts/migration/${MSG_IMPORT_MORSE_ACCOUNTS_PATH}"

# Update the hash after unstaking
./pocketd tx migration import-morse-accounts \
  "tools/scripts/migration/${MSG_IMPORT_MORSE_ACCOUNTS_PATH}" \
  --update-hash-only

# Beautify the JSON file
jq '.' "tools/scripts/migration/${MSG_IMPORT_MORSE_ACCOUNTS_PATH}" > temp.json && mv temp.json "tools/scripts/migration/${MSG_IMPORT_MORSE_ACCOUNTS_PATH}"

# Commit changes
git commit -am "Auto-unstaked validators and updated hash"
git push
```

**Manual Unstaking (Alternative):**

For individual validator groups, use the manual unstake script:

```bash
./tools/scripts/params/manual_unstake.sh \
  "tools/scripts/migration/msg_import_morse_accounts_${MAINNET_SNAPSHOT_HEIGHT}_${MAINNET_SNAPSHOT_DATE}.json" \
  'c409a9e0d1be8780fe0b29dcdf72f8a879fb110c,08e5727cd7fbc4bc97ef3246da7379043f949f70,278654d9daf0e0be2c4e4da5a26c3b4149c5f6d0,81522de7711246fca147a34173dd2a462dc77a5a,c86b27e72c32b64db3eae137ffa84fec007a9062,79cbe645f2b4fa767322faf59a0093e6b73a2383,a86b6a5517630a23aec3dc4e3479a5818c575ac2,882f3f23687a9f3dddf6c65d66e9e3184ca67573,96f2c414b6f3afbba7ba571b7de360709d614e62,05db988509a25dd812dfd1a421cbf47078301a16'
```

Check the diff and update:

```bash
# Clean up backup files
rm "tools/scripts/migration/msg_import_morse_accounts_${MAINNET_SNAPSHOT_HEIGHT}_${MAINNET_SNAPSHOT_DATE}.json.backup."*

# Review changes
git diff .

# Update hash and commit
./pocketd tx migration import-morse-accounts \
  "tools/scripts/migration/msg_import_morse_accounts_${MAINNET_SNAPSHOT_HEIGHT}_${MAINNET_SNAPSHOT_DATE}.json" \
  --update-hash-only

git commit -am "Auto-unstaked Morse validators for entity XXX"
git push
```

Repeat this process for each user with validators to unstake.

#### 3.3 Update migration artifacts in git

Commit the migration artifact files:

```bash
# Commit and push
git commit -am "Added Morse MainNet state export and import message"
git push
```

### 4. Social Consensus

#### 4.1 Distribute Canonical Account State Import Message

Distribute the `msg_import_morse_accounts_${SNAPSHOT_HEIGHT}_${SNAPSHOT_DATE}.json` and its hash for public verification by Morse account/stake-holders.

#### 4.2 Align on Account State via Social Consensus

- Wait for consensus (offchain, time-bounded).
- React to feedback as needed.

### 5. Import Canonical State into Shannon

:::danger

This can **ONLY BE DONE ONCE** on networks with the `allow_morse_account_import_overwrite` param disabled (e.g. MainNet).

:::

**Network-specific Import Commands:**

```bash
# Set common variables
export IMPORT_MSG_PATH="tools/scripts/migration/${MSG_IMPORT_MORSE_ACCOUNTS_PATH}"
export GAS_FLAGS="--gas=auto --gas-prices=1upokt --gas-adjustment=1.5"

# MainNet
## Port forward
kubectl port-forward pods/mainnet-validator1-pocketd-0 26658:26657 9091:9090 -n mainnet

## Run the command
pocketd tx migration import-morse-accounts \
  "$IMPORT_MSG_PATH" \
  --from=grove_mainnet_genesis \
  --home=~/.pocket_prod --keyring-backend=test \
  --node=http://localhost:26658 --chain-id=pocket-alpha \
  $GAS_FLAGS

# Alpha TestNet
## Port forward
kubectl port-forward pods/alpha-validator1-pocketd-0 26658:26657 9091:9090 -n testnet-alpha

## Run the command
pocketd tx migration import-morse-accounts \
  "$IMPORT_MSG_PATH" \
  --from=pnf_alpha \
  --home=~/.pocket_prod --keyring-backend=test \
  --node=http://localhost:26658 --chain-id=pocket-alpha \
  $GAS_FLAGS

# Beta TestNet
## Port forward
kubectl port-forward pods/testnet-beta-validator1-pocketd-0 26658:26657 9091:9090 -n testnet-beta

pocketd tx migration import-morse-accounts "$IMPORT_MSG_PATH" \
  --from=pokt1f0c9y7mahf2ya8tymy8g4rr75ezh3pkklu4c3e \
  --home=~/.pocket_prod --keyring-backend=test \
  --node=http://localhost:26658 --chain-id=pocket-beta \
  $GAS_FLAGS

# LocalNet (for testing)
pocketd tx migration import-morse-accounts \
  "$IMPORT_MSG_PATH" \
  --from=pnf \
  --home=./localnet/pocketd --keyring-backend=test \
  --node=http://localhost:26657 \
  $GAS_FLAGS
```

:::tip kubectl not set up or not cooperatiing?

As an alternative to `kubectl port-forward`, you can use the network-specific public RPC endpoint by removing the `--node` and `--chain-id` flags, and replacing them with the `--network=<local|alpha|beta|main>` flag.

:::

<details>
<summary>Legacy convenience functions for `import-morse-accounts` by network</summary>

```bash
# LocalNet
pocketd tx migration import-morse-accounts "$MSG_IMPORT_MORSE_ACCOUNTS_PATH" --from pnf --home=./localnet/pocketd --network=local --gas=auto --gas-prices=1upokt --gas-adjustment=1.5

# Alpha TestNet
pocketd tx migration import-morse-accounts "$MSG_IMPORT_MORSE_ACCOUNTS_PATH" --from pokt1r6ja6rz6rpae58njfrsgs5n5sp3r36r2q9j04h --home=~/.pocket_prod --network=alpha --gas=auto --gas-prices=1upokt --gas-adjustment=1.5

# Beta TestNet
pocketd tx migration import-morse-accounts "$MSG_IMPORT_MORSE_ACCOUNTS_PATH" --from pokt1f0c9y7mahf2ya8tymy8g4rr75ezh3pkklu4c3e --home=~/.pocket_prod --network=beta --gas=auto --gas-prices=1upokt --gas-adjustment=1.5

# MainNet
pocketd tx migration import-morse-accounts "$MSG_IMPORT_MORSE_ACCOUNTS_PATH" --from pokt18808wvw0h4t450t06uvauny8lvscsxjfyua7vh --home=~/.pocket_prod --network=main --gas=auto --gas-prices=1upokt --gas-adjustment=1.5
```

</details>

### 6. Query Canonical State in Shannon

**Query All Claimable Accounts:**

```bash
pocketd query migration list-morse-claimable-account
```

**Query Specific Account:**

```bash
# Example: Query an auto-unstaked account
export TEST_MORSE_ADDRESS="c409a9e0d1be8780fe0b29dcdf72f8a879fb110c"
pocketd query migration show-morse-claimable-account "$TEST_MORSE_ADDRESS"
```

<details>
<summary>Network-specific query commands</summary>

```bash
# LocalNet
pocketd query migration list-morse-claimable-account --node=http://localhost:26657

# Alpha TestNet
pocketd query migration list-morse-claimable-account --network=alpha

# Beta TestNet
pocketd query migration list-morse-claimable-account --network=beta

# MainNet
pocketd query migration list-morse-claimable-account --network=main
```

</details>

**Validation Commands:**

```bash
# Query an account that was auto-unstaked
export TEST_MORSE_ADDRESS="c409a9e0d1be8780fe0b29dcdf72f8a879fb110c"
pocketd query migration show-morse-claimable-account "$TEST_MORSE_ADDRESS"

# Validate all accounts in the import message (LocalNet example)
pocketd tx migration import-morse-accounts \
  "tools/scripts/migration/msg_import_morse_accounts_${MAINNET_SNAPSHOT_HEIGHT}_${MAINNET_SNAPSHOT_DATE}.json" \
  --from=pnf \
  --home=./localnet/pocketd --keyring-backend=test \
  --node=http://localhost:26657 \
  --gas=auto --gas-adjustment=1.5
```

### 7. Cleanup & Documentation

Document the details of the snapshot upload in [tools/scripts/migration/README.md](https://github.com/pokt-network/poktroll/blob/main/tools/scripts/migration/README.md).

**Update Migration Artifacts:**

```bash
# Document the completed migration
echo "Snapshot Height: $MAINNET_SNAPSHOT_HEIGHT" >> tools/scripts/migration/README.md
echo "Snapshot Date: $MAINNET_SNAPSHOT_DATE" >> tools/scripts/migration/README.md
echo "Import Message: msg_import_morse_accounts_${MAINNET_SNAPSHOT_HEIGHT}_${MAINNET_SNAPSHOT_DATE}.json" >> tools/scripts/migration/README.md

# Final commit
git add tools/scripts/migration/README.md
git commit -m "Updated migration documentation for snapshot ${MAINNET_SNAPSHOT_HEIGHT}"
git push
```

## State Validation: Morse Account Holders

:::info Fun Analogy 👯

- It's like making sure you and your friends (your accounts) are on "the list" before it gets printed out and handed to the crypto-club bouncer.
- Double-check that all names are on the list and spelled correctly; **the bouncer at crypto-club is brutally strict**.

:::

### Why Validate?

**The output in `msg_morse_import_accounts_${SNAPSHOT_HEIGHT}_${DATE}.json` MUST be validated before step 5.**

The [State Transfer Overview](3_state_transfer_overview.md) process determines the _official_ set of claimable Morse accounts (balances/stakes) on Shannon.

It's **critical** for Morse account/stake-holders to confirm their account(s) are included and correct in the proposed `msg_import_morse_claimable_accounts.json`.

### How to Validate

Firstly, **Retrieve** the latest proposed `msg_morse_import_accounts_${SNAPSHOT_HEIGHT}_${DATE}.json` (contains both the state and its hash).

Then, **Validate** using the Shannon CLI like so:

```bash
pocketd tx migration validate-morse-accounts "./msg_import_morse_accounts_${MAINNET_SNAPSHOT_HEIGHT}_${MAINNET_SNAPSHOT_DATE}.json" [morse_hex_address1, ...]
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

If you're getting this error, make sure your `--home` flag points to a Shannon (not Morse) directory:

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
  --network=<network> \
  --gas auto --gas-prices 1upokt --gas-adjustment 1.5 \
  --home=$HOME/.pocket_prod
```

### `http2 frame too large`

If you're seeing the following issue:

```bash
rpc error: code = Unavailable desc = connection error: desc = "error reading server preface: http2: frame too large"
```

The http/grpc configs of the `RPC_ENDPOINT` you're using may need to be configured.

If you're running it yourself in `k8s`, a workaround can be to replace this command:

```bash
pocketd tx migration import-morse-accounts ./tools/scripts/migration/morse_account_state_alpha.json  --from pnf_alpha  --home=$HOME/.pocket_prod --network=alpha --gas=auto --gas-prices=1upokt --gas-adjustment=1.5
```

with

```bash
kubectl port-forward pods/alpha-validator1-pocketd-0 26657:26657 9090:9090 -n testnet-alpha --address 0.0.0.0

pocketd tx migration import-morse-accounts ./tools/scripts/migration/morse_account_state_alpha.json  --from pnf_alpha  --home=$HOME/.pocket_prod --network=alpha --gas=auto --gas-prices=1upokt --gas-adjustment=1.5 --node=localhost:26657
```
