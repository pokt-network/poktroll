---
title: Mnemonic Seed Phrase Recovery
sidebar_position: 3
---

# Recovering an Account from a Mnemonic Seed Phrase <!-- omit in toc -->

:::warning Security Notice

Recovering your wallet with a mnemonic seed phrase requires
you to enter sensitive information. Ensure you are in a secure and private environment
before proceeding.

:::

Losing access to your wallet can be stressful, but if you've backed up your mnemonic
seed phrase, recovering your account is straightforward!

- [Pre-requisites](#pre-requisites)
- [Step 1: Prepare to Recover Your Wallet](#step-1-prepare-to-recover-your-wallet)
- [Step 2: Recovering the Wallet](#step-2-recovering-the-wallet)
- [Step 3: Verify Wallet Recovery](#step-3-verify-wallet-recovery)

## Pre-requisites

- You have the mnemonic seed phrase of the wallet you wish to recover
- `poktrolld` is installed on your system; see the [installation guide](./poktrolld_cli.md) for more details

## Step 1: Prepare to Recover Your Wallet

Before you start, ensure you're in a secure and private environment.
The mnemonic seed phrase is the key to your wallet, and exposing it can lead to loss of funds.

## Step 2: Recovering the Wallet

To recover your wallet, use the `poktrolld keys add` command with the `--recover` flag.
You will be prompted to enter the mnemonic seed phrase and optionally, a BIP39 passphrase if you've set one.

```bash
poktrolld keys add <insert-your-wallet-name-here> --recover
```

Example:

```bash
poktrolld keys add myRecoveredWallet --recover
```

After entering the mnemonic seed phrase, the command will recover your wallet,
displaying the wallet's address and public key.

No mnemonic will be shown since the wallet is being recovered, not created anew.

## Step 3: Verify Wallet Recovery

After recovery, you can use the `poktrolld keys list` command to list all wallets in your keyring.

Verify that the recovered wallet appears in the list with the correct address.

```sh
poktrolld keys list
```

**Congratulations!** You have successfully recovered your Pocket Network wallet!
