---
title: Recovering Morse Accounts
sidebar_position: 15
---

:::danger Authority-gated

The recovery process is authority-gated and requires proper authorization.

Ensure the destination Shannon address is correct.

:::

## Quickstart <!-- omit in toc -->

Recovering an unclaimable Morse account to a Shannon address on Beta:

```bash
pocketd tx migration recover-account <638...> <pokt1...> --from=pnf_beta --network=beta
```

Recovering a Morse module account to a Shannon address on Beta:

```bash
pocketd tx migration recover-account DAO <pokt1...> --from=pnf_beta --network=beta
```

For other options and configurations, run:

```bash
pocketd tx migration recover-account --help
```

## Table of Contents <!-- omit in toc -->

- [Overview](#overview)
  - [Step 1: Verify the account balance and details](#step-1-verify-the-account-balance-and-details)
  - [Step 2: Verify the account can be recovered](#step-2-verify-the-account-can-be-recovered)
  - [Step 3: Import authority address](#step-3-import-authority-address)
  - [Required Authorization](#required-authorization)
  - [Example 1: Recover DAO Module Account](#example-1-recover-dao-module-account)
  - [Example 2: Recover Another Account](#example-2-recover-another-account)
- [Complete Recovery Examples](#complete-recovery-examples)
  - [Copy-Paste Ready Commands](#copy-paste-ready-commands)
- [Hyperlinks to explorers](#hyperlinks-to-explorers)
  - [MainNet Explorers](#mainnet-explorers)
  - [TestNet Explorers](#testnet-explorers)
- [Background Documentation](#background-documentation)
- [Pre-requisites](#pre-requisites)

## Overview

This guide covers how to recover Morse accounts that:

1. Are in the [recovery allowlist](https://github.com/pokt-network/poktroll/blob/main/x/migration/recovery/recovery_allowlist.go)
2. Are unclaimable due to some reason (lost private key, unclaimable, etc)
3. Can only be claimed by an onchain authority (i.e. Pocket Network Foundation)

### Step 1: Verify the account balance and details

Before attempting recovery, you should verify the account balance and details. Check the [Morse mainnet snapshot](https://raw.githubusercontent.com/pokt-network/poktroll/refs/heads/main/tools/scripts/migration/m sg_import_morse_accounts_170616_2025-06-03.json) to see account information.

Example for the DAO module account:

```json
{
  "shannon_dest_address": "",
  "morse_src_address": "dao",
  "unstaked_balance": {
    "denom": "upokt",
    "amount": "319105435973736"
  },
  "supplier_stake": {
    "denom": "upokt",
    "amount": "0"
  },
  "application_stake": {
    "denom": "upokt",
    "amount": "0"
  },
  "claimed_at_height": "0",
  "unstaking_time": "0001-01-01T00:00:00Z"
}
```

### Step 2: Verify the account can be recovered

Verify that the account is on the recoverable accounts allowlist by checking the [recovery allowlist](https://github.com/pokt-network/poktroll/blob/main/x/migration/recovery/recovery_allowlist.go).

An account is eligible for recovery if it meets **both** criteria:

- It is **unclaimable** (cannot be claimed through normal migration)
- It is on the **recoverable accounts allowlist**

### Step 3: Import authority address

Before recovering accounts, you need to import the authority address that has the proper authorization:

```bash
# Import the authority private key (replace with actual key)
pocketd keys add authority-key --recover

# Or import from a specific keyring backend
pocketd keys add authority-key --recover --keyring-backend=os
```

**Note**: The actual private key hex is omitted for security reasons. Obtain the proper authority key from your organization's secure key management system.

### Required Authorization

You **MUST** have an onchain authorization for the message type `pocket.migration.MsgRecoverMorseAccount`. Check existing authorizations with:

```bash
pocketd query authz grants [granter-address] [grantee-address]
```

### Example 1: Recover DAO Module Account

Recover the DAO module account using a key name:

```bash
# Using key name for destination
pocketd tx migration recover-account dao pnf --from=pnf --network=beta

# Using explicit Shannon address
pocketd tx migration recover-account dao pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw --from=pnf --network=beta
```

### Example 2: Recover Another Account

Recover a different account on MainNet with OS keyring:

```bash
# Recover account on MainNet
pocketd tx migration recover-account [morse-address] [shannon-dest] --from=authority-key --network=main --keyring-backend=os

# Example with specific addresses
pocketd tx migration recover-account validator1 pokt1abc123def456ghi789jkl012mno345pqr678stu --from=authority-key --network=main
```

## Complete Recovery Examples

### Copy-Paste Ready Commands

**Beta Network - DAO Recovery:**

```bash
pocketd tx migration recover-account dao pnf --from=pnf --network=beta --gas=auto --gas-adjustment=1.5 --fees=1000upokt
```

**MainNet - DAO Recovery:**

```bash
pocketd tx migration recover-account dao pnf --from=pnf --network=main --keyring-backend=os --gas=auto --gas-adjustment=1.5 --fees=1000upokt
```

**MainNet - Custom Account Recovery:**

```bash
pocketd tx migration recover-account [MORSE_ADDRESS] [SHANNON_DEST] --from=[AUTHORITY_KEY] --network=main --keyring-backend=os --gas=auto --gas-adjustment=1.5 --fees=1000upokt
```

## Hyperlinks to explorers

Monitor your recovery transactions using these block explorers:

### MainNet Explorers

- **POKTScan**: [https://poktscan.com](https://poktscan.com)
- **Shannon Explorer**: [https://shannon.explorers.guru](https://shannon.explorers.guru)

### TestNet Explorers

- **Beta TestNet**: [https://beta.poktscan.com](https://beta.poktscan.com)
- **Beta Explorer**: [https://shannon-testnet.explorers.guru](https://shannon-testnet.explorers.guru)

## Background Documentation

For more detailed information about the recovery process, refer to:

- [Morse Migration Overview](./morse_migration_overview.md)
- [Account Claiming Process](./account_claiming.md)
- [Recovery Allowlist Source](https://github.com/pokt-network/poktroll/blob/main/x/migration/recovery/recovery_allowlist.go)
- [MainNet Snapshot Data](https://raw.githubusercontent.com/pokt-network/poktroll/refs/heads/main/tools/scripts/migration/msg_import_morse_accounts_170616_2025-06-03.json)

## Pre-requisites

**Common Issues:**

1. **Insufficient Authorization**: Ensure your account has the proper authz grant for `pocket.migration.MsgRecoverMorseAccount`
2. **Account Not Recoverable**: Verify the account is both unclaimable and on the allowlist
3. **Key Not Found**: Make sure you've imported the authority key correctly
4. **Network Issues**: Verify you're connected to the correct network (main/beta)

**Getting Help:**

If you encounter issues, check the transaction hash on the appropriate explorer or contact the development team with your transaction details.
