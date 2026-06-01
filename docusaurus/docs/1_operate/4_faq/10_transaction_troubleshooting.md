---
title: Transaction Troubleshooting
sidebar_position: 10
---

## Transaction Troubleshooting <!-- omit in toc -->

Common errors when broadcasting `pocketd` transactions, and how to recover from them.

- [`account sequence mismatch`](#account-sequence-mismatch)
- [`insufficient fees` / `out of gas`](#insufficient-fees--out-of-gas)
- [`account ... not found`](#account--not-found)
- [`key not found` / wrong keyring backend](#key-not-found--wrong-keyring-backend)
- [A transaction "sent with errors"](#a-transaction-sent-with-errors)

---

## `account sequence mismatch`

```text
account sequence mismatch, expected 42, got 41: incorrect account sequence
```

Every account has a monotonically increasing `sequence` (nonce). Each transaction
must use the next expected sequence. This error means the sequence you signed with
does not match what the chain expects — almost always because a **previous
transaction already consumed the sequence** (and likely succeeded), or because two
transactions were signed/broadcast in quick succession before the chain (or your
wallet UI) refreshed.

:::warning Check before re-sending — the first transaction may have succeeded

Do **not** blindly resubmit. Check on-chain state first; a confused resend can result
in a **double-send**.

:::

1. Check the current account state (note the `sequence`):

   ```bash
   pocketd query auth account <your_address> --network=main
   ```

2. Check the balance / recipient to see whether the prior transaction already landed:

   ```bash
   pocketd query bank balances <your_address> --network=main
   ```

3. If the prior transaction succeeded, you are done. If it genuinely failed, re-broadcast.
   The CLI normally fills the sequence automatically; to set it explicitly, use
   `--sequence <N>` (and `--account-number <N>` when offline-signing).

:::danger Do not "fix" this with `--unordered`

`--unordered` changes transaction ordering semantics and is **not** a workaround for a
sequence mismatch. Resolve the actual sequence state instead.

:::

---

## `insufficient fees` / `out of gas`

```text
insufficient fees; got: 1upokt required: 100upokt
out of gas in location: ...; gasWanted: 200000, gasUsed: 210000
```

The transaction did not pay enough fee, or under-estimated gas.

- Let the CLI estimate gas and price it:

  ```bash
  pocketd tx ... --gas=auto --gas-prices=1upokt --gas-adjustment=1.5
  ```

- For `Proof` submissions specifically, the operator must also hold enough balance to
  cover the `proof_submission_fee` (failure to submit a required proof can lead to
  **slashing**). See the [RelayMiner Config → Payable Proof Submissions](../3_configs/4_relayminer_config.md#payable-proof-submissions).

---

## `account ... not found`

```text
account pokt1... not found
```

The address has never appeared on-chain (it has no `account_number` yet). An account
is only created when it **first receives funds** or signs its first transaction.

- Fund the address (faucet on test networks, or a transfer on mainnet), then retry.
- Note a related gotcha for suppliers: an operator account can exist but still lack an
  **on-chain public key** until it signs its first transaction. A supplier whose
  operator has no on-chain public key cannot sign relay responses. See the
  [Supplier cheat sheet → Suppliers staked on behalf of Owners](../1_cheat_sheets/4_supplier_cheatsheet.md#4-suppliers-staked-on-behalf-of-owners).

---

## `key not found` / wrong keyring backend

```text
<name>.info: key not found
```

The key exists in a **different keyring backend** than the one the command is using.
Keys added under `--keyring-backend file` are not visible to `--keyring-backend test`,
and vice versa.

- List keys for the backend you are actually using:

  ```bash
  pocketd keys list --keyring-backend file
  ```

- Make the backend the default to avoid passing the flag every time. See
  [Configuring the keyring backend](../../2_explore/2_account_management/1_pocketd_cli.md#configuring-the-keyring-backend).

---

## A transaction "sent with errors"

Some wallets report a **failed** transaction with a "sent with errors" style message
and a long stack trace. A transaction that fails `CheckTx` (e.g. a sequence mismatch)
is **rejected and not included** — it did not spend funds.

- Read the **first** line of the error (e.g. `account sequence mismatch`,
  `insufficient fees`) — that is the actionable cause; the trailing stack trace is noise.
- Confirm on-chain state (`query auth account`, `query bank balances`) before retrying,
  per the sequence-mismatch guidance above.
