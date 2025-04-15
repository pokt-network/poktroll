---
title: Testing Protocol Upgrades (Local Environment)
sidebar_position: 3
---

:::warning
This document is intended for core protocol developers.

**It assumes you have followed steps 1 through 4 in the [Release Procedure](./2_release_procedure.md).**
:::

## Table of Contents <!-- omit in toc -->

- [Local Upgrade Verification By Example](#local-upgrade-verification-by-example)
- [1. Start a node running the old software (that will listen on the upgrade)](#1-start-a-node-running-the-old-software-that-will-listen-on-the-upgrade)
- [2. Start a node running the new software (from where the upgrade will be issued)](#2-start-a-node-running-the-new-software-from-where-the-upgrade-will-be-issued)
- [3. Prepare the upgrade transaction in `poktroll_old`](#3-prepare-the-upgrade-transaction-in-poktroll_old)
- [4. Submit \& Verify the upgrade transaction in `poktroll_old`](#4-submit--verify-the-upgrade-transaction-in-poktroll_old)
- [5. Observe the upgrade output in `poktroll_old`](#5-observe-the-upgrade-output-in-poktroll_old)
- [6. Stop `poktroll_old` and start `poktroll_new`](#6-stop-poktroll_old-and-start-poktroll_new)
- [7. Sanity Checks](#7-sanity-checks)

### Local Upgrade Verification By Example

The instructions on this page will show how to validate the upgrade from `v0.1.1` to `v0.1.2` by example on `darwin` (macOS) using an `arm64` architecture.

:::warning TODO(@olshansk): Iterate & Improve

Update this page to use `v0.1.2` to `v0.1.3` because the `v0.1.1` tag had a lot of outdated tooling.

Streamline it during the next upgrade & release.

:::

Note that local environments **DO NOT** support `cosmovisor` and automatic upgrades at the moment. `cosmosvisor` doesn't pull the binary from the upgrade Plan's info field.

However, **IT IS NOT NEEDED** to simulate and test the upgrade procedure.

### 1. Start a node running the old software (that will listen on the upgrade)

In one shell, run the following commands to check out the old version (`v0.1.1`)

```bash
git clone git@github.com:pokt-network/poktroll.git poktroll_old
cd poktroll_old
gco v0.1.1
make go_develop ignite_release ignite_release_extract_binaries
./release_binaries/pocket_darwin_arm64 comet unsafe-reset-all && make localnet_regenesis
./release_binaries/pocket_darwin_arm64 start
```

### 2. Start a node running the new software (from where the upgrade will be issued)

In one shell, run the following commands to check out the new version (`v0.1.2`)

```bash
git clone git@github.com:pokt-network/poktroll.git poktroll_new
cd poktroll_new
gco v0.1.2
make go_develop ignite_release ignite_release_extract_binaries
./release_binaries/pocket_darwin_arm64 start
```

### 3. Prepare the upgrade transaction in `poktroll_old`

:::note Nuanced order of operations

Note that this was (likely) already committed to `main` but was not available in the `v0.1.2` tag because it happened afterwards.

:::

Run the following command to prepare the upgrade transaction:

```bash
./tools/scripts/upgrades/prepare_upgrade_tx.sh v0.1.2
```

Update the `height` in `tools/scripts/upgrades/upgrade_tx_v0.1.2_local.json`:

1. Query the height, increment by 20 (arbitrary value), and assign to an environment variable
2. Update the JSON file with the new height
3. Verify the upgrade transaction

You can copy-paste the following to execute all three steps at once:

```bash
# Step 1
CURRENT_HEIGHT=$(./release_binaries/pocket_darwin_arm64 q consensus comet block-latest -o json | jq '.sdk_block.last_commit.height' | tr -d '"')
UPGRADE_HEIGHT=$((CURRENT_HEIGHT + 20))
# Step 2
sed -i.bak "s/\"height\": \"[^\"]*\"/\"height\": \"$UPGRADE_HEIGHT\"/" tools/scripts/upgrades/upgrade_tx_v0.1.2_local.json
# Step 3
cat ./tools/scripts/upgrades/upgrade_tx_v0.1.2_local.json
```

### 4. Submit & Verify the upgrade transaction in `poktroll_old`

Submit the upgrade transaction like so:

```bash
./release_binaries/pocket_darwin_arm64 tx authz exec tools/scripts/upgrades/upgrade_tx_v0.1.2_local.json --yes --from=pnf
```

And verify that the upgrade plan is planned onchain:

```bash
./release_binaries/pocket_darwin_arm64 query upgrade plan
```

### 5. Observe the upgrade output in `poktroll_old`

Once the `UPGRADE_HEIGHT` is reached, you should see the following output containing `ERR` in your terminal:

```bash
4:33PM ERR UPGRADE "v0.1.2" NEEDED at height: 30: ...
```

The validator should stop working.

### 6. Stop `poktroll_old` and start `poktroll_new`

In `poktroll_old`, stop the (non-functional) validator (i.e.`cmd + c` on `darwin`).

In `poktroll_new`, start the (presumably-functional) validator:

```bash
./release_binaries/pocket_darwin_arm64 start
```

:::warning Expertise required

If the new validator does not start, this will require expert protocol development hands-on debugging.

:::

### 7. Sanity Checks

Verify that the new validator is on the latest version like so:

```bash
curl -s http://localhost:26657/abci_info | jq '.result.response.version'
```

Which should output `v0.1.2`.

:::warning TODO: Business logic

Query the node for business logic changes

:::
