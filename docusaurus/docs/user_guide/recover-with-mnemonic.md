---
title: Recover using Mnemonic Seed Phrase
---

# Recovering an Account from a Mnemonic Seed Phrase

Losing access to your wallet can be stressful, but if you've backed up your mnemonic seed phrase, recovering your account is straightforward. This guide will walk you through the process of restoring access to your poktrolld blockchain wallet using your mnemonic seed phrase.

:::warning

**Security Notice:** Recovering your wallet with a mnemonic seed phrase requires you to enter sensitive information. Ensure you are in a secure and private environment before proceeding.

:::

## Pre-requisites

- The mnemonic seed phrase of the wallet you wish to recover.
- poktrolld installed on your system. If you haven't installed poktrolld yet, refer to the installation guide.

## Step 1: Prepare to Recover Your Wallet

Before you start, ensure you're in a secure and private environment. The mnemonic seed phrase is the key to your wallet, and exposing it can lead to loss of funds.

## Step 2: Recovering the Wallet

To recover your wallet, use the `poktrolld keys add` command with the `--recover` flag. You will be prompted to enter the mnemonic seed phrase and optionally, a BIP39 passphrase if you've set one.

```sh
poktrolld keys add <wallet-name> --recover
```

Replace `<wallet-name>` with the name you want to assign to the recovered wallet. After running the command, you'll be prompted to enter your mnemonic seed phrase.

### Example:

```sh
poktrolld keys add recoveredWallet --recover
```

After entering the mnemonic seed phrase, the command will recover your wallet, displaying the wallet's address and public key. No mnemonic will be shown since the wallet is being recovered, not created anew.

## Step 3: Verify Wallet Recovery

After recovery, you can use the `poktrolld keys list` command to list all wallets in your keyring. Verify that the recovered wallet appears in the list with the correct address.

```sh
poktrolld keys list
```

Congratulations! You have successfully recovered your wallet on the poktrolld blockchain. Remember to keep your mnemonic phrase secure and follow the best practices for managing your wallet.