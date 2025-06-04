---
title: Recovering Morse Accounts
sidebar_position: 15
---

:::danger Authority-gated

The recovery process is authority-gated and requires proper authorization.

Ensure the destination Shannon address is correct.

:::

## Quickstart <!-- omit in toc -->

Recovering a lost Morse account to a misc Shannon address on Beta:

```bash
pocketd tx migration recover-account A7BEC93013FA51339DE2F62CB0466550C67092F2 <pokt1...> --from=pnf_beta --network=beta
```

Recovering the Morse DAO module account to Shannon's PNF address on Beta:

```bash
pocketd tx migration recover-account DAO pokt1f0c9y7mahf2ya8tymy8g4rr75ezh3pkklu4c3e --from=pnf_beta --network=beta
```

For other options and configurations, run:

```bash
pocketd tx migration recover-account --help
```

## Table of Contents <!-- omit in toc -->

- [Overview](#overview)
- [Step 1: Verify the account balance and details](#step-1-verify-the-account-balance-and-details)
- [Optional Verification Steps](#optional-verification-steps)
- [Step 2: Recover the Account](#step-2-recover-the-account)
- [Step 3: Verify the balance on the recovered account](#step-3-verify-the-balance-on-the-recovered-account)

## Overview

This guide covers how to recover Morse accounts that:

1. Are in the [recovery allowlist](https://github.com/pokt-network/poktroll/blob/main/x/migration/recovery/recovery_allowlist.go)
2. Are unclaimable due to some reason (lost private key, unclaimable, etc)
3. Can only be claimed by an onchain authority (i.e. Pocket Network Foundation)

## Step 1: Verify the account balance and details

Using `A7BEC93013FA51339DE2F62CB0466550C67092F2` as an example on Beta TestNet.

1. Visit [shannon-beta.trustsoothe.io/migration?address=A7..F2](https://shannon-beta.trustsoothe.io/migration?address=5EED...)
2. Look for `A7..F2` in [recovery_allowlist.go](https://github.com/pokt-network/poktroll/blob/main/x/migration/recovery/recovery_allowlist.go).
3. Verify `A7..F2` is in [state export from Morse](https://raw.githubusercontent.com/pokt-network/poktroll/refs/heads/main/tools/scripts/migration/morse_state_export_170616_2025-06-03.json).

## Optional Verification Steps

<details>

<summary>Adding and verifying authority address</summary>

**Add `pnf_beta` to your keyring**:

```bash
pocketd keys import-hex pnf_beta <private-key-hex-for-pnf-beta> --key-type secp256k1 --keyring-backend os
```

Get the address of `pnf_beta`:

```bash
pocketd keys show pnf_beta -a --keyring-backend=os
# pokt1f0c9y7mahf2ya8tymy8g4rr75ezh3pkklu4c3e
```

Verify that `pnf_beta` has the proper authorization:

```bash
pocketd q authz grants-by-grantee pokt1f0c9y7mahf2ya8tymy8g4rr75ezh3pkklu4c3e -o json --network=beta | jq '.grants[] | select(.authorization.value.msg == "/pocket.migration.MsgRecoverMorseAccount")'
```

</details>

## Step 2: Recover the Account

```bash
pocketd tx migration recover-account A7BEC93013FA51339DE2F62CB0466550C67092F2 pokt132y5nzs4xahqy6cmzankn8mn4ec897j50wuzhr --from=pnf_beta --network=beta --keyring-backend=os --gas=auto --gas-adjustment=1.5 --fees=1000upokt
```

You can check the status of the transaction using:

```bash
pocketd q tx --type=hash <transaction-hash> --network=beta
```

## Step 3: Verify the balance on the recovered account

```bash
pocketd q bank balances pokt132y5nzs4xahqy6cmzankn8mn4ec897j50wuzhr --network=beta
```
