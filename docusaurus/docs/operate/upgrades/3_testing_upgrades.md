---
title: Testing Protocol Upgrades (Local Environment)
sidebar_position: 3
---

:::warning
**For core protocol developers only!**

- Complete steps 1–4 in [Release Procedure](./2_release_procedure.md) before starting.
  :::

## 📠🍝 Local Upgrade Testing – Step-by-Step

**This is a copy/paste-ready checklist for validating protocol upgrades locally.**

- Every step is numbered and must be completed in order.
- All commands are ready to copy/paste.
- If you get stuck, ask for help.

---

## Table of Contents

- [📠🍝 Local Upgrade Testing – Step-by-Step](#-local-upgrade-testing--step-by-step)
- [Table of Contents](#table-of-contents)
- [0. Notes \& Prerequisites](#0-notes--prerequisites)
- [1. Start node with old version](#1-start-node-with-old-version)
- [2. Start node with new version](#2-start-node-with-new-version)
- [3. Prepare the upgrade transaction](#3-prepare-the-upgrade-transaction)
- [4. Submit \& verify the upgrade transaction](#4-submit--verify-the-upgrade-transaction)
- [5. Observe the upgrade output](#5-observe-the-upgrade-output)
- [6. Stop old node \& start new node](#6-stop-old-node--start-new-node)
- [7. Sanity checks](#7-sanity-checks)

---

## 0. Notes & Prerequisites

- Example upgrade: `v0.1.1` → `v0.1.2` (macOS, arm64)
- Local environments **do not** support `cosmovisor`/automatic upgrades. This is fine for testing.
- Update this doc for each new upgrade (see TODOs).

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

- In `poktroll_old`, run:

```bash
./tools/scripts/upgrades/prepare_upgrade_tx.sh v0.1.2
```

- Update the `height` in `tools/scripts/upgrades/upgrade_tx_v0.1.2_local.json`:

```bash
# Get current height, add 20 (buffer)
CURRENT_HEIGHT=$(./release_binaries/pocket_darwin_arm64 q consensus comet block-latest -o json | jq '.sdk_block.last_commit.height' | tr -d '"')
UPGRADE_HEIGHT=$((CURRENT_HEIGHT + 20))
sed -i.bak "s/\"height\": \"[^\"]*\"/\"height\": \"$UPGRADE_HEIGHT\"/" tools/scripts/upgrades/upgrade_tx_v0.1.2_local.json
cat ./tools/scripts/upgrades/upgrade_tx_v0.1.2_local.json
```

---

## 4. Submit & verify the upgrade transaction

- In `poktroll_old`, submit:

```bash
./release_binaries/pocket_darwin_arm64 tx authz exec tools/scripts/upgrades/upgrade_tx_v0.1.2_local.json --yes --from=pnf
```

- Verify onchain:

```bash
./release_binaries/pocket_darwin_arm64 query upgrade plan
```

---

## 5. Observe the upgrade output

- When `UPGRADE_HEIGHT` is reached, you should see output like:

```bash
ERR UPGRADE "v0.1.2" NEEDED at height: <height>: ...
```

- The validator should stop.

---

## 6. Stop old node & start new node

- In `poktroll_old`, stop the validator (`cmd + c` on macOS).
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
