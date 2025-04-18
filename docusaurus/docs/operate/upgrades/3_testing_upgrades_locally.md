---
title: Protocol Upgrades Local Testing
sidebar_position: 3
---

:::warning
**For core protocol developers only!**

Make sure to complete steps 1–4 in [Release Procedure](./2_release_procedure.md) before starting.

:::

## 📠🍝 Testing Protocol Upgrades Locally: Step-by-Step <!-- omit in toc -->

**This contains step-by-step instructions for testing & validating protocol upgrades locally.**

- Every step is numbered and must be completed in order.
- All commands are ready to copy/paste.
- If you get stuck, ask for help.
- If you find improvements, please update this doc.

---

## Table of Contents <!-- omit in toc -->

- [0. Prerequisite Notes](#0-prerequisite-notes)
- [1. Start node with old version](#1-start-node-with-old-version)
- [2. Start node with new version](#2-start-node-with-new-version)
- [3. Prepare the upgrade transaction](#3-prepare-the-upgrade-transaction)
- [4. Submit \& verify the upgrade transaction](#4-submit--verify-the-upgrade-transaction)
- [5. Observe the upgrade output](#5-observe-the-upgrade-output)
- [6. Stop old node \& start new node](#6-stop-old-node--start-new-node)
- [7. Sanity checks](#7-sanity-checks)
- [Pro Mode](#pro-mode)

---

## 0. Prerequisite Notes

- Local environments **do not** support `cosmovisor`/automatic upgrades. This is fine for testing.
- This document shows how to test an upgrade locally using the `pocket_darwin_arm64` binary on a `macOS` machine with an `arm64` architecture. Adapt this to your environment.
- The example shows how to test an upgrade from `v0.1.1` → `v0.1.2` assuming both of them already exist. Adapt this to your needs.

---

## 1. Start node with old version

```bash
git clone git@github.com:pokt-network/poktroll.git poktroll_old
cd poktroll_old
gco v0.1.1
make go_develop ignite_release ignite_release_extract_binaries
./release_binaries/pocket_darwin_arm64 comet unsafe-reset-all && make localnet_regenesis
./release_binaries/pocket_darwin_arm64 start
```

---

## 2. Start node with new version

```bash
git clone git@github.com:pokt-network/poktroll.git poktroll_new
cd poktroll_new
gco v0.1.2
make go_develop ignite_release ignite_release_extract_binaries
./release_binaries/pocket_darwin_arm64 start
```

---

## 3. Prepare the upgrade transaction

In `poktroll_old`, the following file should already exist if you have completed the [Release Procedure](2_release_procedure.md):

```bash
tools/scripts/upgrades/upgrade_tx_v0.1.2_local.json
```

It was created by running:

```bash
./tools/scripts/upgrades/prepare_upgrade_tx.sh v0.1.2
```

Open up a second shell in `poktroll_old` and update the `height` in `tools/scripts/upgrades/upgrade_tx_v0.1.2_local.json`:

```bash
# Get current height, add 20 (buffer)
CURRENT_HEIGHT=$(./release_binaries/pocket_darwin_arm64 q consensus comet block-latest -o json | jq '.sdk_block.last_commit.height' | tr -d '"')
UPGRADE_HEIGHT=$((CURRENT_HEIGHT + 20))
sed -i.bak "s/\"height\": \"[^\"]*\"/\"height\": \"$UPGRADE_HEIGHT\"/" tools/scripts/upgrades/upgrade_tx_v0.1.2_local.json
cat ./tools/scripts/upgrades/upgrade_tx_v0.1.2_local.json
```

---

## 4. Submit & verify the upgrade transaction

In `poktroll_old`, submit the upgrade transaction:

```bash
./release_binaries/pocket_darwin_arm64 tx authz exec tools/scripts/upgrades/upgrade_tx_v0.1.2_local.json --yes --from=pnf
```

Verify it was submitted onchain:

```bash
./release_binaries/pocket_darwin_arm64 query upgrade plan
```

---

## 5. Observe the upgrade output

When `UPGRADE_HEIGHT` is reached, you should see output like:

```bash
ERR UPGRADE "v0.1.2" NEEDED at height: <height>: ...
```

**🛑 The validator should stop 🛑**

---

## 6. Stop old node & start new node

- In `poktroll_old`, stop the validator (`cmd/ctrl + c` on macOS).
- In `poktroll_new`, start the validator:

```bash
./release_binaries/pocket_darwin_arm64 start
```

:::warning
If the new validator does not start, expert debugging is required.
:::

---

## 7. Sanity checks

- Check the new validator version:

```bash
curl -s http://localhost:26657/abci_info | jq '.result.response.version'
```

- Should output: `v0.1.2`

:::note
Query the node for business logic changes as needed.
:::

## Pro Mode

```
Old repo:
1. Checkout out latest release
2. Followed instrucitons
3. Ensure it’s running

New repo:
1. Checked out new branch
2. Updated allUpgrades
3. Created new upgrades/.go file
    1. Added the changes I need
4. Ran the new make target
5. Drafted a new release
    1. Wait for release artifacts to run: https://github.com/pokt-network/poktroll/actions
    2. Binaries will get attached
    3. Get the tag

Old repo:
1. Generate transactionw with the old release: ./tools/scripts/upgrades/prepare_upgrade_tx.sh v0.1.4-test1 —test
    1. Make sure to include the testing flag
2. Update the height
3. CRITICAL:
    1.  cp tools/scripts/upgrades/upgrade_tx_v0.1.4-test1_local.json ../poktroll/tools/scripts/upgrades
4. Submit the tx
5. Follow logs

New repo:
1. Update “v0.1.4” to “"v0.1.4-test1" in v0.1.4.go
2. Compile the binaries
3. Run the binary


The loop:
1. New repo
    1. Update business logic
    2. Update upgrade
    3. Start recompiling
2. Old repo
    1. Optionall: Update code & recompile
    2. Run regeneis
    3. Start the miner
    4. Update height
    5. Submit upgrade
    6. Query upgrade
    7. Wait for it to break
3. New repo
    1. Start the relay miner
    2. Observe logs
    3. Validate upgrade


Use `ignite_release_local` instead of `ignite_release`


—

Quick turnaround

Old repo:
1. Shell 1:
    1. ./release_binaries/pocket_darwin_arm64 comet unsafe-reset-all && make localnet_regenesis
    2. ./release_binaries/pocket_darwin_arm64 start
2. Shell 2:
    1. Get height and update -> local.json
    2. ./release_binaries/pocket_darwin_arm64 tx authz exec tools/scripts/upgrades/upgrade_tx_v0.1.4-test1_local.json --yes --from=pnf

New repo:

1. make go_develop ignite_release_local ignite_release_extract_binaries
2. ./release_binaries/pocket_darwin_arm64 start
3. Logic for sucstom validation of your changes
```
