---
title: State Transfer Playbook
sidebar_position: 4
---

ETVL stands for Export -> Transform -> Validate -> Load.

## Table of Contents <!-- omit in toc -->

- [High-Level Account State Transfer Playbook for Authority (i.e. Foundation)](#high-level-account-state-transfer-playbook-for-authority-ie-foundation)
- [High-Level State Validation Playbook for Morse Account holders](#high-level-state-validation-playbook-for-morse-account-holders)

### High-Level Account State Transfer Playbook for Authority (i.e. Foundation)

:::warning TODO_MAINNET: This playbook is a WIP

This playbook is an early WIP and will be updated and moved elsewhere once the process is finalized.

:::

1. **Export** the canonical `MorseStateExport` from the Morse network:

   ```bash
   pocket util export-genesis-for-reset <published canonical export height> pocket > morse_state_export.json
   ```

2. **Transform** the `MorseStateExport` into the proposed canonical `MorseAccountState`:

   ```bash
   pocketd migrate collect-morse-accounts morse_state_export.json morse_account_state.json
   ```

3. **Distribute** the `MorseAccountState` and its hash for verification by Morse account/stake-holders.
4. **Wait for consensus** after an offchain time-bounded period on the `MorseAccountState`, reacting to any offchain feedback, as necessary.
5. **Load** (i.e. import) the canonical `MorseAccountState` on Shanno

   ```bash
   pocketd tx migrate import-morse-accounts morse_account_state.json --from <authorized-key-name> --grpc-addr=<shannon-network-grpc-endpoint>
   ```

   :::important
   The `--grpc-addr` flag expects the gRPC endpoint for the Shannon network you intend to import the Morse accounts into (e.g. DevNet, TestNet, etc.). See: Shannon RPC Endpoints for more info.
   :::

:::danger TODO_MAINNET: Select snapshot height

Replace `<published canonical export height>` with the actual height once known.

:::

### High-Level State Validation Playbook for Morse Account holders

:::info Fun Analogy 👯

It's like making sure you and your friends (your accounts) are on "the list" before it gets printed out and handed to the crypto-club bouncer.

You'd be wise to double-check that all the names are on the list and are spelled correctly; **the bouncer at crypto-club is brutally strict**.

:::

Since the result of the ETVL process effectively determines the canonical set of claimable Morse accounts (and their balances/stakes) on Shannon,
it is critical that Morse account/stake-holders confirm that the proposed `MsgImportMorseClaimableAccounts` includes an accurate representation of their account(s).

Morse account/stake-holders who wish to participate in the social consensus process for the "canonical" `MorseAccountState` can do so by:

1. **Downloading the proposed `MsgImportMorseClaimableAccounts`**: this encapsulates both the `MorseAccountState` and its hash

   :::warning TODO_MAINNET
   Link to latest published [`MsgImportMorseClaimableAccounts`](https://github.com/pokt-network/poktroll/blob/main/proto/pocket/migration/tx.proto#L50) proposal.
   :::

2. **Use the Shannon CLI to validate the proposed `MsgImportMorseClaimableAccounts`**: See `./msg_import_morse_claimable_accounts.json` as an example.

   ```bash
   # TODO_MAINNET_CRITICAL(@bryanchriswhite, #1034): Complete this example once the CLI is available.
   pocketd tx migration validate-morse-accounts ./msg_import_morse_claimable_accounts.json [morse_hex_address1, ...]
   ```

   :::note

   Multiple Morse addresses MAY be passed to the `validate-morse-accounts` command.
   For each, the corresponding `MorseClaimableAccount` is printed for the purpose of manual inspection and validation.

   :::
