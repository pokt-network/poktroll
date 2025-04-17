---
title: Claiming Morse Accounts
sidebar_position: 6
---

- [Account Definition](#account-definition)
- [How do I claim my Morse POKT?](#how-do-i-claim-my-morse-pokt)
  - [1. Prerequisite - Create a new Shannon key](#1-prerequisite---create-a-new-shannon-key)
  - [2. Claim Shannon Claim Tokens](#2-claim-shannon-claim-tokens)
  - [3. Submit Claim Transaction](#3-submit-claim-transaction)
- [How does it work?](#how-does-it-work)

## Account Definition

This page describes how to claim a Morse "Account" on Shannon.

This covers accounts which:

- **ARE NOT** staked as an Application
- **ARE NOT** staked as a Supplier
- **DO NOT** have any POKT staked
- **DO** have a non-zero POKT balance

## How do I claim my Morse POKT?

### 1. Prerequisite - Create a new Shannon key

For example, running the following command:

```bash
pocketd tx migration claim-account \
  ./pocket-account-8b257c7f4e884e49bafc540d874f33f91436e1dc.json \
  --from app1
```

### 2. Claim Shannon Claim Tokens

### 3. Submit Claim Transaction

Should prompt for a passphrase and produce output similar to the following:

```shell
Enter Decrypt Passphrase:
MsgClaimMorseAccount {
  "shannon_dest_address": "pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4",
  "morse_src_address": "8B257C7F4E884E49BAFC540D874F33F91436E1DC",
  "morse_signature": "hLGhLRjP6jgP6wgOIaYFxIxT3z4jb4IBDKovMkX5AqUsOqdF+rEIO5aofOKnmYW9BkqL0v2DfUfE3nj25FNhBA=="
}
Confirm MsgClaimMorseAccount: y/[n]:
```

## How does it work?

Claiming an unstaked account will mint the unstaked balance of the Morse account being claimed to the Shannon account which the signer of the `MsgClaimMorseAccount`.

This unstaked balance amount is retrieved from the corresponding onchain `MorseClaimableAccount` which was imported by the foundation.
