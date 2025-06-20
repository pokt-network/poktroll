---
title: Claiming Morse Accounts
sidebar_position: 6
---

- [Account Definition](#account-definition)
- [How do I claim my Morse POKT?](#how-do-i-claim-my-morse-pokt)
  - [0. Prerequisites](#0-prerequisites)
  - [1. Export your Morse `keyfile.json`](#1-export-your-morse-keyfilejson)
  - [2. Create a new Shannon key](#2-create-a-new-shannon-key)
  - [3. Create your onchain Shannon account](#3-create-your-onchain-shannon-account)
  - [4. Ensure your Shannon account exists onchain](#4-ensure-your-shannon-account-exists-onchain)
  - [5. Check your claimable Morse account](#5-check-your-claimable-morse-account)
  - [6. Claim your Morse Pocket](#6-claim-your-morse-pocket)
  - [7. Verify your Shannon balance](#7-verify-your-shannon-balance)
- [Troubleshooting](#troubleshooting)
  - [Transaction signing errors](#transaction-signing-errors)
  - [Onchain Fee Requirement](#onchain-fee-requirement)

## Account Definition

This page describes how to claim a Morse "Account" on Shannon.

This covers accounts which:

- **DO** have a non-zero POKT balance
- **DO NOT** have any POKT staked
- **ARE NOT** staked as an Application
- **ARE NOT** staked as a Supplier

## How do I claim my Morse POKT?

### 0. Prerequisites

You have read the introduction in [Claiming Introduction](./5_claiming_introduction.md) and ensure:

- You have installed the Morse `pocket` CLI
- You have installed the Shannon `pocketd` CLI
- You have imported your Morse key into a keyring
- You have a valid RPC endpoint

### 1. Export your Morse `keyfile.json`

Export your `keyfile.json` from the Morse keyring to `pocket-account-<morse-address>.json` like so:

```bash
pocket accounts export <morse-address>
```

Or specify a custom path:

```bash
pocket accounts export <morse-address> --path <custom-path>.json
```

:::tip Import your Morse account first

If you have your private key hex and want to import your Morse account first, you can do it like so

```bash
pocket accounts import-raw <priv-key-hex>
```

:::

:::warning Remember your Encrypt Passphrase

When exporting the account, you will be prompted with `Enter Encrypt Passphrase`. Please be sure to remember or write it down!

:::

### 2. Create a new Shannon key

```bash
pocketd keys add <your_shannon_key_name>
```

### 3. Create your onchain Shannon account

If you're using a newly generated key/account, then you will need to use one of the (network-specific) community faucets to trigger onchain account creation.

<!-- TODO(@bryanchriswhite): Add a link once available! -->

For testnets, you can send yourself either uPOKT or MACT ("Morse Account Claimer Token").
**Only 1 of EITHER minimal denomination** is sufficient to create the onchain account, such that it is ready to be used for claiming.

:::warning Mainnet "Faucet"

The only token available via Mainnet faucet(s) is MACT.

:::

<details>
<summary>Why do we need MACT?</summary>

MACT is needed to enable users to claim their Morse accounts using new Shannon accounts.
These new accounts must exist onchain before they can be used, which requires a transaction to create them.
MACT provides a simple and dedicated way to do this.
A public faucet will distribute MACT so users can prepare their accounts without relying on other tokens, making the claiming process smooth and accessible.

```mermaid
---
title: Design Constraint / Effect Causal Flowchart
---
flowchart

nka(new key algo):::constraint
nacc(**user** MUST generate new account):::effect
nasn(no initial onchain account sequence number):::constraint
noregen(can't use regensis):::constraint

claim(MUST claim Morse accounts):::effect

nka --> nacc
nacc -->|user-initiated == cannot be predicted/pre-computed| noregen

noregen --> claim
noregen --> nasn

cacc("MUST 'create' onchain account (additional Tx)"):::effect

nasn --> cacc

nt("new 'Morse Account Claimer Token' (MACT)")
f(MACT faucet)

cacc --> nt
cacc --> f
f -.-> nt

mac(Morse account/actor claim protocol)

claim --> mac

classDef constraint color:#f00,stroke:#f00
classDef effect color:#f80,stroke:#f80
```

</details>

<!--TODO_MAINNET : add mainnet MACT faucet link-->

Use one of the following faucets:

- [Alpha Testnet MACT/POKT Faucet](https://faucet.alpha.testnet.pokt.network/)
- [Beta Testnet MACT/POKT Faucet](https://faucet.beta.testnet.pokt.network/)
- [Mainnet MACT Faucet](https://faucet.pocket.network/)

:::tip Grove Employees 🌿

If you're a Grove Employee, you can use the helpers [here](https://www.notion.so/buildwithgrove/Playbook-Streamlining-rc-helpers-for-Shannon-Alpha-Beta-Main-Network-Environments-152a36edfff680019314d468fad88864?pvs=4) to fund the account like so:

```bash
# Get the Shannon address
pocketd keys show <your_shannon_key_name> -a

# Fund the Shannon key with 1MPOKT
pkd_<NETWORK>fund_pokt <your_shannon_address>
# OR
# Fund the Shannon key with 1MACT
pkd_<NETWORK>fund_mact <your_shannon_address>
```

:::

### 4. Ensure your Shannon account exists onchain

```bash
# Ensure this returns a nonzero balance
pocketd query bank balances <your_shannon_key_name> --network=<network> #e.g. local, alpha, beta, main

```

### 5. Check your claimable Morse account

:::tip Ensure your Morse Address is ALL CAPS

For this next step, you will need to convert your Morse address from lower case to ALL CAPS. You can ask an AI to perform this for you.

:::

```bash
pocketd query migration show-morse-claimable-account \
  <morse-address-ALL-CAPS> \
  --network=<network> #e.g. local, alpha, beta, main
```

### 6. Claim your Morse Pocket

Running the following command:

```bash
pocketd tx migration claim-account \
  pocket-account-<morse-keyfile-export>.json \
  --from=<your_shannon_key_name> \
  --network=<network> #e.g. local, alpha, beta, main
```

The above will prompt for your generated Morse Encrypt Passphrase and produce output similar to the following:

```bash
Enter Decrypt Passphrase:
MsgClaimMorseAccount {
  "shannon_dest_address": "pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4",
  "morse_src_address": "8B257C7F4E884E49BAFC540D874F33F91436E1DC",
  "morse_signature": "hLGhLRjP6jgP6wgOIaYFxIxT3z4jb4IBDKovMkX5AqUsOqdF+rEIO5aofOKnmYW9BkqL0v2DfUfE3nj25FNhBA=="
}
Confirm MsgClaimMorseAccount: y/[n]:
```

:::tip Be patient. Don't Panic!

This step may sit in your terminal for a minute or so. Be patient and don't panic, it is working!

:::

### 7. Verify your Shannon balance

```bash
pocketd query bank balances <your_shannon_address> --network=<network> #e.g. local, alpha, beta, main
```

## Troubleshooting

### Transaction signing errors

If you're hitting errors related to signature verification, ensure you've specified
the following flags based on your environment and keyring config

- `--network`: one of `local`, `alpha`, `beta`, `main`
- `--home`: the path to your keyring directory
- `--keyring-backend`: one of `test`, `file`, `os`, `kwallet`, `pass`, `keosd`

### Onchain Fee Requirement

```bash
pocketd query migration params --home=~/.pocketd --network=<network> #e.g. local, alpha, beta, main
```

```yaml
params:
  waive_morse_claim_gas_fees: true
```
