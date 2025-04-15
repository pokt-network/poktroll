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
- [5. Observe the upgrade output](#5-observe-the-upgrade-output)
- [6. Verify Node Software](#6-verify-node-software)
- [7. (If Applicable) Test Consensus Breaking Changes](#7-if-applicable-test-consensus-breaking-changes)

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

And verify that the upgrade plan is onchain:

```bash
./release_binaries/pocket_darwin_arm64 query upgrade plan
```

### 5. Observe the upgrade output

1. **(`new` repo)** - Observe the output:

   - A successful upgrade should output `applying upgrade "v0.2" at height: 20 module=x/upgrade`.
   - The node on the new version should continue producing blocks.
   - If there were errors during the upgrade, investigate and address them.

### 6. Verify Node Software

### 7. (If Applicable) Test Consensus Breaking Changes
