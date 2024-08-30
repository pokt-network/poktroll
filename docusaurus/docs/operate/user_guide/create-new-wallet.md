---
title: Create a New Wallet
sidebar_position: 1
---

# Create a New Wallet <!-- omit in toc -->

:::warning Security Notice

**ALWAYS back up your key and/or mnemonic**. Store it in a secure
location accessible only to you, such as a password manager, or written down
in a safe place. Under your 🛏️ does not count!

:::

This guide will walk you through creating a new wallet on the Pocket Network.

- [What is a keyring backend?](#what-is-a-keyring-backend)
- [Step 1: Install poktrolld](#step-1-install-poktrolld)
- [Step 2: Creating the Wallet](#step-2-creating-the-wallet)
- [Step 3: Backing Up Your Wallet](#step-3-backing-up-your-wallet)

## What is a keyring backend?

Before proceeding, it's critical to understand the implications of keyring backends
for securing your wallet.

By default, `--keyring-backend=test` is used for demonstration
purposes in this documentation, suitable for initial testing.

In production, operators should consider using a more secure keyring backend
such as `os`, `file`, or `kwallet`. For more information on keyring backends,
refer to the [Cosmos SDK Keyring documentation](https://docs.cosmos.network/main/user/run-node/keyring).

## Step 1: Install poktrolld

Ensure you have `poktrolld` installed on your system.

Follow the [installation guide](./install) specific to your operating system.

## Step 2: Creating the Wallet

To create a new wallet, use the `poktrolld keys add` command followed by your
desired wallet name. This will generate a new address and mnemonic phrase for your wallet.

```bash
poktrolld keys add <insert-your-desired-wallet-name-here>
```

After running the command, you'll receive an output similar to the following:

```plaintext
- address: pokt1beef420
  name: myNewWallet
  pubkey: '{"@type":"/cosmos.crypto.secp256k1.PubKey","key":"A31T7iUyr6SwT5Wyy3BNgRqlObq3FqYpW4cTAkfE+6c2"}'
  type: local


**Important** write this mnemonic phrase in a safe place.
It is the only way to recover your account if you ever forget your password.

your seed mnemonic phase here
```

## Step 3: Backing Up Your Wallet

After creating your wallet, **YOU MUST** back up your mnemonic phrase. This phrase
is the key to your wallet, and losing it means losing access to your funds.

Here are some tips for securely backing up your mnemonic phrase:

- Write it down on paper and store it in multiple secure locations.
- Consider using a password manager to store it digitally, ensuring the service is reputable and secure.
- Avoid storing it in plaintext on your computer or online services prone to hacking.

**Congratulations!** You have successfully created a new wallet on Pocket Network.
