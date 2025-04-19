---
title: State Transfer Playbook
sidebar_position: 4
---

This page is intended for the Foundation (Authority) or whoever is managing the state transfer process.

## Table of Contents <!-- omit in toc -->

- [1. Retrieve a Pruned Morse Snapshot](#1-retrieve-a-pruned-morse-snapshot)
- [2. Export Morse Snapshot State](#2-export-morse-snapshot-state)
- [3. Transform Morse Export to a Canonical Account State](#3-transform-morse-export-to-a-canonical-account-state)
- [4. Distribute Canonical Account State](#4-distribute-canonical-account-state)
- [5. Align on Account State via Social Consensus](#5-align-on-account-state-via-social-consensus)
- [6. Import Canonical State into Shannon](#6-import-canonical-state-into-shannon)
- [7. Query Canonical State in Shannon](#7-query-canonical-state-in-shannon)
- [State Validation: Morse Account Holders](#state-validation-morse-account-holders)
  - [Why Validate?](#why-validate)
  - [How to Validate](#how-to-validate)
- [Troubleshooting](#troubleshooting)
  - [I don't have a real snapshot on my machine](#i-dont-have-a-real-snapshot-on-my-machine)
  - [`invalid character at start of key`](#invalid-character-at-start-of-key)
  - [`failed to get grant with given granter: ..., grantee: ... & msgType: /pocket.migration.MsgImportMorseClaimableAccounts`](#failed-to-get-grant-with-given-granter--grantee---msgtype-pocketmigrationmsgimportmorseclaimableaccounts)
  - [`http2 frame too large`](#http2-frame-too-large)

---

## 1. Retrieve a Pruned Morse Snapshot

Go to [Liquify's Snapshot Explorer](https://pocket-snapshot-uk.liquify.com/#/pruned/) and download the latest pruned snapshot.

Export the snapshot into a new directory on your local machine.

```bash
mkdir -p $HOME/morse-snapshot
# 1. Untar the snapshot file
tar -xvf <snapshot-file>.tar -C $HOME/morse-snapshot
# 2. Change directory to the extracted snapshot folder
$HOME/morse-snapshot
# 3. Create the data directory
mkdir data
# 4. Move all *.db files to the data directory
mv *.db data
```

**⚠️ Note the height and date of the snapshot⚠️**

## 2. Export Morse Snapshot State

Choose the snapshot height, which must be less than or equal to the snapshot height retrieved above. **This will be the published canonical export height.**

```bash
pocket util export-genesis-for-reset <HEIGHT> pocket --datadir $HOME/morse-snapshot > morse_state_export.json
```

## 3. Transform Morse Export to a Canonical Account State

```bash
pocketd tx migration collect-morse-accounts morse_state_export.json morse_account_state.json
```

## 4. Distribute Canonical Account State

Distribute the `morse_account_state.json` and its hash for public verification by Morse account/stake-holders.

## 5. Align on Account State via Social Consensus

- Wait for consensus (offchain, time-bounded).
- React to feedback as needed.

## 6. Import Canonical State into Shannon

:::danger

This can **ONLY BE DONE ONCE** per network.

:::

The following `import-morse-accounts` command can be used to import the canonical account state into Shannon:

```bash
pocketd tx migration \
  import-morse-accounts morse_account_state.json \
  --from <authorized-key-name> \
  --grpc-addr=<shannon-network-grpc-endpoint> \
  --home <shannon-home-directory> \
  --chain-id=<shannon-chain-id> \
  --gas=auto --gas-prices=1upokt --gas-adjustment=1.5
```

<details>
<summary>Convenience functions for `import-morse-accounts` by network</summary>

```bash
# LocalNet
pocketd tx migration import-morse-accounts morse_account_state.json --from pnf --grpc-addr=localhost:9090 --home=./localnet/pocketd --chain-id=pocket --gas=auto --gas-prices=1upokt --gas-adjustment=1.5

# Alpha TestNet
pocketd tx migration import-morse-accounts morse_account_state.json --from pokt1r6ja6rz6rpae58njfrsgs5n5sp3r36r2q9j04h --home=~/.pocket_prod --chain-id=pocket-alpha --gas=auto --gas-prices=1upokt --gas-adjustment=1.5 --node=http://shannon-grove-rpc.alpha.poktroll.com --grpc-addr=https://shannon-grove-grpc.alpha.poktroll.com

# Beta TestNet
  pocketd tx migration import-morse-accounts morse_account_state.json --from pokt1f0c9y7mahf2ya8tymy8g4rr75ezh3pkklu4c3e --home=~/.pocket_prod --chain-id=pocket-beta --gas=auto --gas-prices=1upokt --gas-adjustment=1.5 --node=http://shannon-grove-rpc.beta.poktroll.com --grpc-addr=https://shannon-grove-grpc.beta.poktroll.com

# MainNet
pocketd tx migration import-morse-accounts morse_account_state.json --from pokt18808wvw0h4t450t06uvauny8lvscsxjfyua7vh --home=~/.pocket_prod --chain-id=pocket-mainnet --gas=auto --gas-prices=1upokt --gas-adjustment=1.5 --node=http://shannon-grove-rpc.mainnet.poktroll.com --grpc-addr=https://shannon-grove-grpc.mainnet.poktroll.com
```

</details>

## 7. Query Canonical State in Shannon

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

## State Validation: Morse Account Holders

:::warning TODO(@bryanchriswhite): Incomplete

Show how to generate `msg_import_morse_claimable_accounts.json`.

:::

:::info Fun Analogy 👯

- It's like making sure you and your friends (your accounts) are on "the list" before it gets printed out and handed to the crypto-club bouncer.
- Double-check that all names are on the list and spelled correctly; **the bouncer at crypto-club is brutally strict**.

:::

### Why Validate?

**The output in `morse_account_state.json` MUST be validated before step 6.**

The [State Transfer Overview](3_state_transfer_overview.md) process determines the _official_ set of claimable Morse accounts (balances/stakes) on Shannon.

It's **critical** for Morse account/stake-holders to confirm their account(s) are included and correct in the proposed `msg_import_morse_claimable_accounts.json`.

### How to Validate

Firstly, **Retrieve** the latest proposed `msg_import_morse_claimable_accounts.json` (contains both the state and its hash).

Then, **Validate** using the Shannon CLI like so:

```bash
pocketd tx migration validate-morse-accounts ./msg_import_morse_claimable_accounts.json [morse_hex_address1, ...]
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
pocketd tx migration import-morse-accounts ./tools/scripts/migration/morse_account_state_alpha.json  --from pnf_alpha  --home=$HOME/.pocket_prod --chain-id=pocket-alpha --gas=auto --gas-prices=1upokt --gas-adjustment=1.5 --grpc-addr=shannon-testnet-grove-grpc.alpha.poktroll.com:443 --node=https://shannon-testnet-grove-rpc.alpha.poktroll.com
```

with

```bash

kubectl port-forward pods/alpha-validator1-pocketd-0 26657:26657 9090:9090 -n testnet-alpha --address 0.0.0.0

pocketd tx migration import-morse-accounts ./tools/scripts/migration/morse_account_state_alpha.json  --from pnf_alpha  --home=$HOME/.pocket_prod --chain-id=pocket-alpha --gas=auto --gas-prices=1upokt --gas-adjustment=1.5 --grpc-addr=localhost:9090 --node=localhost:26657
```
