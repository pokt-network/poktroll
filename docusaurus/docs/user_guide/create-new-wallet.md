---
title: Create new wallet
sidebar_position: 1
---

# Creating a New Wallet

This guide will walk you through creating a new wallet on the Pocket Network Shannon blockchain. Before proceeding, it's important to understand the implications of keyring backends for securing your wallet. By default, `--keyring-backend=test` is used for demonstration purposes in this documentation, suitable for initial testing. However, for actual deployments or production use, operators should consider using a more secure keyring backend, such as `os`, `file`, or `kwallet`. For more information on keyring backends, refer to the [Cosmos SDK Keyring documentation](https://docs.cosmos.network/main/user/run-node/keyring).

:::info

**Security Notice:** Always back up your key/mnemonic. Store it in a secure location accessible only to you, such as a password manager, or written down and kept in a safe.

:::

## Step 1: Install poktrolld

Ensure you have poktrolld installed on your system. Follow the [installation guide](./install-poktrolld) specific to your operating system.

## Step 2: Creating the Wallet

To create a new wallet, use the `poktrolld keys add` command followed by your desired wallet name. This will generate a new address and mnemonic phrase for your wallet.

```sh
poktrolld keys add <wallet-name>
```

Replace `<wallet-name>` with your desired wallet name.

### Example:

```sh
poktrolld keys add myNewWallet
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

After creating your wallet, you'll be given a mnemonic phrase. This phrase is the key to your wallet, and losing it means losing access to your funds. Here are some tips for securely backing up your mnemonic phrase:

- Write it down on paper and store it in multiple secure locations.
- Consider using a password manager to store it digitally, ensuring the service is reputable and secure.
- Avoid storing it in plaintext on your computer or online services prone to hacking.

Congratulations! You have successfully created a new wallet on the poktrolld blockchain. Remember to keep your mnemonic phrase secure and follow the best practices for managing your new wallet.
