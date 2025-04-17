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
- [Pro](#pro)

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

## Pro

migrate local testing notes here
