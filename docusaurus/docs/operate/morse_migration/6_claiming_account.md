---
title: Claiming Morse Accounts
sidebar_position: 6
---

- [Account Definition](#account-definition)
- [How do I claim my Morse POKT?](#how-do-i-claim-my-morse-pokt)
  - [0. Prerequisites](#0-prerequisites)
  - [1. Export your Morse `keyfile.json`](#1-export-your-morse-keyfilejson)
  - [2. Create a new Shannon key](#2-create-a-new-shannon-key)
  - [3. Fund your Shannon account](#3-fund-your-shannon-account)
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

:::tip Import it first

If you have your private key hex and want to import it first, you can do it like so

```bash
pocket accounts import-raw <priv-key-hex>
```

:::

### 2. Create a new Shannon key

```bash
pocketd keys add <your_shannon_key_name>
```

### 3. Fund your Shannon account

You need to make sure the public key exists onchain and has funding to send the claim transaction.

Use one of the following faucets:

- [Alpha Testnet](https://faucet.alpha.testnet.pokt.network/)
- [Beta Testnet](https://faucet.beta.testnet.pokt.network/)
- Mainnet: Coming soon.

:::tip Grove Employees 🌿

If you're a Grove Employee, you can use the helpers [here](https://www.notion.so/buildwithgrove/Playbook-Streamlining-rc-helpers-for-Shannon-Alpha-Beta-Main-Network-Environments-152a36edfff680019314d468fad88864?pvs=4) to fund the account like so:

```bash
# Get the Shannon address
pocketd keys show <your_shannon_key_name> -a

# Fund the Shannon key
pkd_<NETWORK>fund <your_shannon_address>
```

:::

### 4. Ensure your Shannon account exists onchain

```bash
pocketd query account <shannon-dest-address> --node=${RPC_ENDPOINT}
```

### 5. Check your claimable Morse account

```bash
pocketd query migration show-morse-claimable-account \
  <morse-address-all-caps> \
  --node=${RPC_ENDPOINT}
```

### 6. Claim your Morse Pocket

Running the following command:

```bash
pocketd tx migration claim-account \
  pocket-account-<morse-keyfile-export>.json \
  --from=<your_shannon_address> \
  --node=${RPC_ENDPOINT} --chain-id=pocket-<network> \
  --home=~/.pocketd --keyring-backend=test --no-passphrase
  # --gas=auto --gas-prices=1upokt --gas-adjustment=1.5
```

Should prompt for a passphrase and produce output similar to the following:

```bash
Enter Decrypt Passphrase:
MsgClaimMorseAccount {
  "shannon_dest_address": "pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4",
  "morse_src_address": "8B257C7F4E884E49BAFC540D874F33F91436E1DC",
  "morse_signature": "hLGhLRjP6jgP6wgOIaYFxIxT3z4jb4IBDKovMkX5AqUsOqdF+rEIO5aofOKnmYW9BkqL0v2DfUfE3nj25FNhBA=="
}
Confirm MsgClaimMorseAccount: y/[n]:
```

### 7. Verify your Shannon balance

```bash
pocketd query bank balances <shannon-dest-address> --node=${RPC_ENDPOINT}
```

## Troubleshooting

### Transaction signing errors

If you're hitting errors related to signature verification, ensure you've specified
the following flags based on your environment and keyring config

- `--chain-id`: one of `pocket-alpha`, `pocket-beta`, `pocket`
- `--home`: the path to your keyring directory
- `--keyring-backend`: one of `test`, `file`, `os`, `kwallet`, `pass`, `keosd`

### Onchain Fee Requirement

```bash
pocketd query migration params --node=${RPC_ENDPOINT} --home=~/.pocketd
```

```yaml
params:
  waive_morse_claim_gas_fees: true
```
