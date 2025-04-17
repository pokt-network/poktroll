---
title: State Transfer Playbook
sidebar_position: 4
---

This page is intended for the Foundation (Authority) or whoever is managing the state transfer process.

## Table of Contents <!-- omit in toc -->

- [1. Retrieve a Pruned Morse Snapshot](#1-retrieve-a-pruned-morse-snapshot)
- [2. Export Morse State](#2-export-morse-state)
- [3. Transform Export to Canonical Account State](#3-transform-export-to-canonical-account-state)
- [4. Distribute Account State](#4-distribute-account-state)
- [5. Align on Social Consensus](#5-align-on-social-consensus)
- [6. Import Canonical State into Shannon](#6-import-canonical-state-into-shannon)
- [State Validation: Morse Account Holders](#state-validation-morse-account-holders)
  - [Why Validate?](#why-validate)
  - [How to Validate](#how-to-validate)

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

### 3. Transform Export to Canonical Account State

```bash
pocketd tx migration collect-morse-accounts morse_state_export.json morse_account_state.json
```

:::warning Troubleshooting

Ensure your `--home` flag points to a Shannon directory if you see this error:

```bash
failed to read in /Users/olshansky/.pocket/config/config.toml: While parsing config: toml: invalid character at start of key: {
```

:::

### 4. Distribute Account State

Distribute the `morse_account_state.json` and its hash for public verification by Morse account/stake-holders.

### 5. Align on Social Consensus

- Wait for consensus (offchain, time-bounded).
- React to feedback as needed.

### 6. Import Canonical State into Shannon

```bash
pocketd tx migration \
  import-morse-accounts morse_account_state.json \
  --from <authorized-key-name> \
  --grpc-addr=<shannon-network-grpc-endpoint> \
  --home <shannon-home-directory> \
  --chain-id=<shannon-chain-id> \
  --gas=auto --gas-prices=1upokt --gas-adjustment=1.5
```

For example, to run the above on different networks (replacing `--home`):

```bash
# LocalNet
pocketd tx migration import-morse-accounts morse_account_state.json --from pnf --grpc-addr=localhost:9090 --home=./localnet/pocketd --chain-id=pocket --gas=auto --gas-prices=1upokt --gas-adjustment=1.5

# Alpha TestNet
pocketd tx migration import-morse-accounts morse_account_state.json --from <authorized-key-name> --grpc-addr=https://shannon-grove-grpc.alpha.poktroll.com --home=~/.pocket_prod --chain-id=pocket-alpha --gas=auto --gas-prices=1upokt --gas-adjustment=1.5

# Beta TestNet
pocketd tx migration import-morse-accounts morse_account_state.json --from <authorized-key-name> --grpc-addr=https://shannon-grove-grpc.beta.poktroll.com --home=~/.pocket_prod --chain-id=pocket-beta --gas=auto --gas-prices=1upokt --gas-adjustment=1.5

# MainNet
pocketd tx migration import-morse-accounts morse_account_state.json --from <authorized-key-name> --grpc-addr=https://shannon-grove-grpc.mainnet.poktroll.com --home=~/.pocket_prod --chain-id=pocket-mainnet --gas=auto --gas-prices=1upokt --gas-adjustment=1.5
```

### State Validation: Morse Account Holders

**The output in `morse_account_state.json` MUST be validated before step 6.**

:::info Fun Analogy 👯

- It's like making sure you and your friends (your accounts) are on "the list" before it gets printed out and handed to the crypto-club bouncer.
- Double-check that all names are on the list and spelled correctly; **the bouncer at crypto-club is brutally strict**.

:::

#### Why Validate?

- The [ETVL](3_state_transfer_overview.md) process determines the _official_ set of claimable Morse accounts (balances/stakes) on Shannon.
- It's **critical** for Morse account/stake-holders to confirm their account(s) are included and correct in the proposed `msg_import_morse_claimable_accounts.json`.

#### How to Validate

Firstly, **Retrieve** the latest proposed `msg_import_morse_claimable_accounts.json` (contains both the state and its hash).

Then, **Validate** using the Shannon CLI like so:

```bash
pocketd tx migration validate-morse-accounts ./msg_import_morse_claimable_accounts.json [morse_hex_address1, ...]
```

- You can pass multiple Morse addresses to the command
- For each address, the corresponding `MorseClaimableAccount` is printed for manual inspection and validation
