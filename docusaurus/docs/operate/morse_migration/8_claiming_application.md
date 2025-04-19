---
title: Claiming Morse Applications
sidebar_position: 8
---

## Table of Contents <!-- omit in toc -->

- [What is this?](#what-is-this)
- [How do I claim my Morse Application as a Shannon Application?](#how-do-i-claim-my-morse-application-as-a-shannon-application)
  - [0. Prerequisites](#0-prerequisites)
  - [1. Prepare your Morse and Shannon Keys and Accounts](#1-prepare-your-morse-and-shannon-keys-and-accounts)
  - [2. Prepare your application config](#2-prepare-your-application-config)
  - [3. Claim your Morse Application](#3-claim-your-morse-application)
  - [4. Example output](#4-example-output)
  - [5. Verify your Shannon application](#5-verify-your-shannon-application)
  - [6. What happened?](#6-what-happened)
- [Troubleshooting](#troubleshooting)

## What is this?

- Claim your Morse Application as a Shannon Application
- This is like staking a new Shannon Application, but you **don't specify `stake_amount`**
- All config is the same as [staking an application](../configs/app_staking_config.md) **except** omit `stake_amount`

## How do I claim my Morse Application as a Shannon Application?

### 0. Prerequisites

- You have read the [Claiming Introduction](./5_claiming_introduction.md)
- You have installed the Morse `pocket` CLI
- You have installed the Shannon `pocketd` CLI
- You have imported your Morse key into a keyring
- You have a valid RPC endpoint
- You are familiar with how to stake a native Shannon Application (see [application staking config](../configs/app_staking_config.md))

### 1. Prepare your Morse and Shannon Keys and Accounts

Follow steps 1-5 from [Claiming Morse Account](./6_claiming_account.md)

### 2. Prepare your application config

Use the same format as for staking an application. See [Application staking config](../configs/app_staking_config.md) for details.

Make sure to **omit `stake_amount`**.

:::danger CRITICAL: Omit `stake_amount`

- **DO NOT** include `stake_amount` in your application config when claiming
- If you include it, the claim will fail

:::

### 3. Claim your Morse Application

```bash
pocketd tx migration claim-application \
  pocket-account-<morse-keyfile-export>.json \
  <service_id> \
  --from=<your_shannon_address> \
  --node=${RPC_ENDPOINT} --chain-id=pocket-<network> \
  --home=~/.pocketd --keyring-backend=test --no-passphrase
# --gas=auto --gas-prices=1upokt --gas-adjustment=1.5 (optional)
```

### 4. Example output

```shell
Enter Decrypt Passphrase:
MsgClaimMorseApplication {
  "shannon_dest_address": "pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4",
  "morse_src_address": "1A0BB8623F40D2A9BEAC099A0BAFDCAE3C5D8288",
  "morse_signature": "6kax1TKdvP1sIGrz8lW8jH/jQxv5OiPiFq0/BG5sEfLwVyVNVXihDhJNXd0cQtwDiMPB88PCkvWZOdY/WMY4Dg==",
  "service_config": {
    "service_id": "anvil"
  }
}
Confirm MsgClaimMorseApplication: y/[n]: y
```

### 5. Verify your Shannon application

```bash
pocketd query application <your_shannon_address> --node=${RPC_ENDPOINT}
```

### 6. What happened?

- **Unstaked balance** of Morse account is minted to your Shannon account
- **Stake** is set on Shannon using the onchain Morse application's stake amount
- Both values come from the onchain `MorseClaimableAccount`
- Shannon actors can modify staking configurations (e.g. `service_id`)

## Troubleshooting

See: `pocketd tx migration claim-application --help` for more details.
