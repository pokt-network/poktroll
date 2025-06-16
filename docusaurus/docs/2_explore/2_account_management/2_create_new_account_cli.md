---
title: Create a New Account (CLI)
sidebar_position: 2
---

:::warning Security Notice

⚠️ **ALWAYS back up your key and/or mnemonic after creating a new account** ⚠️

Store it in a secure location accessible only to you, such as a password manager,
or written down in a safe place.

:::

## Quickstart tl;dr <!-- omit in toc -->

**Add a new wallet**:

```bash
pocketd keys add $USER
```

**Retrieve the address**:

```bash
pocketd keys show $USER -a
```

## Table of Contents <!-- omit in toc -->

This guide will walk you through creating a new wallet on the Pocket Network.

- [Prerequisites: Install `pocketd`](#prerequisites-install-pocketd)
- [Exporting \& Importing Hex Private Keys](#exporting--importing-hex-private-keys)
- [Creating a new wallet Wallet](#creating-a-new-wallet-wallet)
- [Backing Up Your Wallet](#backing-up-your-wallet)
- [🔑 HD Derivation Path](#-hd-derivation-path)
- [Keyring Backends](#keyring-backends)
  - [Keyring Directory Behavior: `--home`, `--keyring-backend`, and `--keyring-dir`](#keyring-directory-behavior---home---keyring-backend-and---keyring-dir)

## Prerequisites: Install `pocketd`

Ensure you have `pocketd` installed on your system.

Follow the [installation guide](1_pocketd_cli.md) specific to your operating system.

## Exporting & Importing Hex Private Keys

You can import a hex private key into your keyring like so:

```bash
pocketd keys import-hex <wallet_name> <hex_private_key>
```

And export a hex private key from your keyring like so:

```bash
pocketd keys export <wallet_name> --unsafe --unarmored-hex --yes
```

For more details, see:

```bash
pocketd keys --help
```

## Creating a new wallet Wallet

To create a new wallet, use the `pocketd keys add` command followed by your
desired wallet name. This will generate a new address and mnemonic phrase for your wallet.

```bash
pocketd keys add <insert-your-desired-wallet-name-here>
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

## Backing Up Your Wallet

After creating your wallet, **YOU MUST** back up your mnemonic phrase. This phrase
is the key to your wallet, and losing it means losing access to your funds.

Here are some tips for securely backing up your mnemonic phrase:

- Write it down on paper and store it in multiple secure locations.
- Consider using a password manager to store it digitally, ensuring the service is reputable and secure.
- Avoid storing it in plaintext on your computer or online services prone to hacking.

## 🔑 HD Derivation Path

`pocketd` supports [BIP-0044](https://github.com/bitcoin/bips/blob/master/bip-0044.mediawiki)-compatible HD wallets, with `POKT` registered under `coin_type = 635` (`path_component = 0x8000027b`). This assignment is defined in [SLIP-0044](https://github.com/satoshilabs/slips/blob/master/slip-0044.md).

The default derivation path used is:

```bash
m/44'/635'/0'/0/0
```

To use this path, run:

```bash
pocketd keys add --coin-type=635
```

You can view additional options with:

```bash
pocketd keys add --help
```

**References:**

- **BIP-0044**: [bitcoin/bips/blob/master/bip-0044.mediawiki](https://github.com/bitcoin/bips/blob/master/bip-0044.mediawiki)
- **SLIP-0044**: [satoshilabs/slips/blob/master/slip-0044.md](https://github.com/satoshilabs/slips/blob/master/slip-0044.md)

## Keyring Backends

Before proceeding, it's critical to understand the implications of keyring backends
for securing your wallet.

By default, `--keyring-backend=test` is used for demonstration
purposes in this documentation, suitable for initial testing.

In production, operators should consider using a more secure keyring backend
such as `os`, `file`, or `kwallet`. For more information on keyring backends,
refer to the [Cosmos SDK Keyring documentation](https://docs.cosmos.network/main/user/run-node/keyring).

### Keyring Directory Behavior: `--home`, `--keyring-backend`, and `--keyring-dir`

In the Cosmos SDK (and thus in `pocketd`):

- `--home` sets the root directory for app state (default: `~/.pocket`)
- `--keyring-backend` sets how keys are stored (`os`, `file`, `test`, `memory`)
- `--keyring-dir` overrides where keys are stored, but still nests by backend

**Example:**

```bash
pocketd keys list --home=. --keyring-backend=test --keyring-dir=./foo
```

This creates:

```bash
./foo/keyring-test/
```

So `--keyring-dir` works, but the backend (e.g. `test`) decides the final subfolder. That’s why you see `foo/keyring-test`.

This creates the keyring directory inside of the path you provide to `--keyring-dir`, with a subfolder corresponding to the backend you choose.
