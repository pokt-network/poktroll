---
title: Password-less Keyring
sidebar_position: 6
---

## Setting up a password-less `poktrolld` <!-- omit in toc -->

:::danger No password

These instructions are intended to streamline usage of `poktrolld` on Debian
machines to **AVOID** providing a password each time.

**Only follow these instructions if you know what you're doing.**
:::

## Table of Contents <!-- omit in toc -->

- [Prerequisites](#prerequisites)
- [Background](#background)
- [Instructions](#instructions)
  - [1. Install `pass` (password store utility)](#1-install-pass-password-store-utility)
  - [2. Create a GPG Key](#2-create-a-gpg-key)
  - [3. Find Your GPG Key ID](#3-find-your-gpg-key-id)
  - [4. Initialize pass with your GPG key ID](#4-initialize-pass-with-your-gpg-key-id)
  - [5. Store Cosmos Keyring Password](#5-store-cosmos-keyring-password)
  - [6. Verify Password Storage](#6-verify-password-storage)
  - [7. Test Configuration](#7-test-configuration)

## Prerequisites

1. You are running any Shannon service on a `Debian` machine.
2. You have installed the [poktrolld CLI](./poktrolld_cli.md).
3. You have created a `poktroll` user following one of the guides in the docs.
4. ⚠️ You are annoyed about having to enter your password every time ⚠️

## Background

`poktrolld` uses the Cosmos SDK keyring. For details on how it works, and understanding
what a `backend` is, see [the official docs](https://docs.cosmos.network/v0.46/run-node/keyring.html).

This document will focus on how to use `poktrolld` with the `os` backend without
a password on a Debian machine, and assume you have read the Cosmos documentation.

## Instructions

### 1. Install `pass` (password store utility)

```bash
sudo apt-get install pass
```

### 2. Create a GPG Key

Generate a new GPG key pair - you'll be prompted for:

- Kind of key: Choose RSA
- Key size: 3072 bits is recommended
- Key validity: Choose your preferred duration
- Your name and email

```bash
gpg --full-generate-key
```

### 3. Find Your GPG Key ID

List your secret keys and find your key ID.

```bash
gpg --list-secret-keys --keyid-format LONG
```

The output will look like:

```bash
sec rsa3072/B9448E560E033C02 <-- This is your key ID
5F79E46574CF39CDA4FB46BDB9448E560E033C02
uid [ultimate] Your Name <your.email@example.com>
```

### 4. Initialize pass with your GPG key ID

Replace `B9448E560E033C02` with your actual key ID from the step abpve

```bash
pass init B9448E560E033C02
```

### 5. Store Cosmos Keyring Password

Store your password - you will be prompted to enter it.

```bash
pass insert cosmos-keyring
```

### 6. Verify Password Storage

This will display your stored password

```bash
pass cosmos-keyring
```

### 7. Test Configuration

Test if poktrolld can now access the keyring without prompting

```bash
poktrolld keys list
```

:::warning
Note: Make sure to keep your GPG private key secure, as it's used to decrypt your stored passwords.
:::
