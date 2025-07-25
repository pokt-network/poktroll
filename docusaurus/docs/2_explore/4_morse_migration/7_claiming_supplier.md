---
title: Claiming Morse Suppliers
sidebar_position: 7
---

## Table of Contents <!-- omit in toc -->

- [What is this?](#what-is-this)
- [How do I claim my Morse Supplier as a Shannon Supplier?](#how-do-i-claim-my-morse-supplier-as-a-shannon-supplier)
  - [0. Prerequisites](#0-prerequisites)
  - [1. Prepare your Morse and Shannon Keys and Accounts](#1-prepare-your-morse-and-shannon-keys-and-accounts)
  - [2. Prepare your supplier config](#2-prepare-your-supplier-config)
  - [3. Claim your Morse Supplier](#3-claim-your-morse-supplier)
  - [4. Example output](#4-example-output)
  - [5. Verify your Shannon supplier](#5-verify-your-shannon-supplier)
  - [6. Ensure your Shannon Supplier has an onchain public key](#6-ensure-your-shannon-supplier-has-an-onchain-public-key)
    - [How to check if you have an onchain account](#how-to-check-if-you-have-an-onchain-account)
  - [7. What happened above?](#7-what-happened-above)
- [Troubleshooting](#troubleshooting)

## What is this?

- Claim your Morse Supplier as a Shannon Supplier
- This is like staking a new Shannon Supplier, but you **don't specify `stake_amount`**
- All config is the same as [staking a supplier](../../1_operate/3_configs/3_supplier_staking_config.md) **except** omit `stake_amount`

## How do I claim my Morse Supplier as a Shannon Supplier?

### 0. Prerequisites

- You have read the [Claiming Introduction](./5_claiming_introduction.md)
- You have installed the Morse `pocket` CLI
- You have installed the Shannon `pocketd` CLI
- You have imported your Morse key into a keyring
- You have a valid RPC endpoint
- You are familiar with how to stake a native Shannon Supplier (see [supplier staking config](../../1_operate/3_configs/3_supplier_staking_config.md))

### 1. Prepare your Morse and Shannon Keys and Accounts

Follow steps 1-5 from [Claiming Morse Account](./6_claiming_account.md)

### 2. Prepare your supplier config

Use the same format as for staking a supplier. See [Supplier staking config](../../1_operate/3_configs/3_supplier_staking_config.md) for details.

Make sure to **omit `stake_amount`**.

:::danger CRITICAL: Omit `stake_amount`

- **DO NOT** include `stake_amount` in your supplier config when claiming
- If you include it, the claim will fail

:::

:::warning OPTIONAL: `non-custodial` staking

- You **can** specify different `owner` and `operator` addresses in your config
- `operator` signs claims/proofs, `owner` controls stake/rewards
- See: [Supplier staking config > Staking types](../../1_operate/3_configs/3_supplier_staking_config.md#staking-types)

:::

### 3. Claim your Morse Supplier

```bash
pocketd tx migration claim-supplier \
  <morse-node-address> \
  pocket-account-<morse-keyfile-export>.json \
  <path-to-your-supplier-config>.yaml \
  --from=<your_shannon_address> \
  --network=<network> \
  --home=~/.pocketd --keyring-backend=test --no-passphrase
# --gas=auto --gas-prices=1upokt --gas-adjustment=1.5 (optional)
```

:::important

`<morse-node-address>` is the hex-encoded address of the Morse node account to be claimed.

`<morse-keyfile-export>` is the path to the Morse private key for ONE OF THE FOLLOWING:

- The Morse node account (i.e. operator); aka `custodial`
- The Morse output account (i.e. owner); aka `non-custodial`

See: `pocketd tx migration claim-supplier --help` for more Shannon details.
See: `pocket nodes --help` for more Morse details.

:::

### 4. Example output

```shell
Enter Decrypt Passphrase:
MsgClaimMorseSupplier {
  "shannon_owner_address": "pokt1chn2mglfxqcp52znqk8jq2rww73qffxczz3jph",
  "shannon_operator_address": "pokt1chn2mglfxqcp52znqk8jq2rww73qffxczz3jph",
  "morse_src_address": "44892C8AB52396BA016ADDD0221783E3BD29A400",
  "morse_signature": "rYyN2mnjyMMrYdDhuw+Hrs98b/svn38ixdSWye3Gr66aAJ9CQhdiaYB8Lta8tiwWIteIM8rmWYUh0QkNdpkNDQ==",
  "services": [
    {
      "service_id": "anvil",
      "endpoints": [
        {
          "url": "http://relayminer1:8545",
          "rpc_type": 3
        }
      ],
      "rev_share": [
        {
          "address": "pokt1chn2mglfxqcp52znqk8jq2rww73qffxczz3jph",
          "rev_share_percentage": 100
        }
      ]
    }
  ]
}
Confirm MsgClaimMorseSupplier: y/[n]: y
```

### 5. Verify your Shannon supplier

```bash
pocketd query supplier <your_shannon_address> --network=<network> #e.g. local, alpha, beta, main
```

### 6. Ensure your Shannon Supplier has an onchain public key

:::warning MUST READ FOR ALL SUPPLIER

Your RelayMiner & Supplier will not work if you don't follow this section.

:::

**Problem**: Migrated Suppliers WITHOUT onchain public keys CANNOT sign Relay Responses and will be **sanctioned by PATH**.

**Root Cause**: A migrated supplier has an onchain account, but it does not have an onchain public key until it signs an onchain transaction.

**Solution**: Submit any onchain transaction where `--from` is the supplier address.

Any transaction will work. For example, a small transfer

```bash
pocketd tx bank send <your_supplier_address> <some_address_you_own> 1upokt --from=<your_supplier_address> ...
```

#### How to check if you have an onchain account

```bash
pocketd q auth account <your_supplier_address> --network=<network> --grpc-insecure=false
```

Account without public key:

```yaml
account:
  type: /cosmos.auth.v1beta1.BaseAccount
  value:
    account_number: "..."
    address: pokt1...
    sequence: "..."
```

Account with public key:

```yaml
account:
  type: /cosmos.auth.v1beta1.BaseAccount
  value:
    account_number: "..."
    address: pokt1...
    public_key:
      type: /cosmos.crypto.secp256k1.PubKey
      value: Ap/Nr...
    sequence: "..."
```

### 7. What happened above?

- **Unstaked balance** of Morse account is minted to your Shannon account
- **Stake** is set on Shannon using the onchain Morse supplier's stake amount
- Both values come from the onchain `MorseClaimableAccount`

## Troubleshooting

See: `pocketd tx migration claim-supplier --help` for more details.
