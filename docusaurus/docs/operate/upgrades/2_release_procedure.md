---
title: Protocol Upgrade Release Procedure (Idiot-Proof)
sidebar_position: 2
---

:::warning
**This guide is for core protocol developers.**

- If you are not comfortable with git, GitHub releases, or scripting, STOP and get help.
- Read [When is a Protocol Upgrade Warranted?](./1_protocol_upgrades.md#when-is-an-protocol-upgrade-warranted) before starting.

:::

## Protocol Upgrade Release – Step-by-Step

**This is a complete, 📠-🍝-ready checklist for releasing protocol upgrades.**

- Every step is numbered and must be completed in order.
- All commands are ready to copy/paste.
- If you get stuck or something fails, see the Troubleshooting section at the end.

---

## Table of Contents

- [Protocol Upgrade Release – Step-by-Step](#protocol-upgrade-release--step-by-step)
- [Table of Contents](#table-of-contents)
- [0. Prerequisites \& Sanity Checks](#0-prerequisites--sanity-checks)
- [1. Ensure `ConsensusVersion` is updated](#1-ensure-consensusversion-is-updated)
- [2. Prepare a New Upgrade Plan](#2-prepare-a-new-upgrade-plan)
- [3. Create a GitHub Release](#3-create-a-github-release)
- [4. Write an Upgrade Transaction (JSON file)](#4-write-an-upgrade-transaction-json-file)
- [5. Validate the Upgrade Binary URLs (Live Network Only)](#5-validate-the-upgrade-binary-urls-live-network-only)
- [6. Test the New Release](#6-test-the-new-release)
- [7. Submit the Upgrade Onchain (Alpha)](#7-submit-the-upgrade-onchain-alpha)
- [8. Update the `homebrew-tap` Formula](#8-update-the-homebrew-tap-formula)
- [9. Submit the Upgrade on Beta \& MainNet](#9-submit-the-upgrade-on-beta--mainnet)
- [10. Troubleshooting \& Canceling an Upgrade](#10-troubleshooting--canceling-an-upgrade)
- [Before You Finish](#before-you-finish)

---

## 0. Prerequisites & Sanity Checks

Before you start:

- [ ] You have push/publish access to the repo and [GitHub releases](https://github.com/pokt-network/poktroll/releases)
- [ ] You have the following CLI tools: `git`, `make`, `jq`, `sed`, `curl`, `go`, `brew`, `pocketd`, etc.
- [ ] You have reviewed [previous upgrades](https://github.com/pokt-network/poktroll/tree/main/app/upgrades) for reference
- [ ] You have read the full [Protocol Upgrade Introduction](./1_protocol_upgrades.md)
- [ ] You understand the difference between `state-breaking` and `consensus-breaking` changes
- [ ] You know how to testyour changes locally (see [Testing Upgrades](./3_testing_upgrades.md))

---

## 1. Ensure `ConsensusVersion` is updated

**⚠️ DO NOT PROCEED until you have completed this ⚠️**

- Bump the `ConsensusVersion` for all modules with `state-breaking` changes.
- This requires manual inspection and understanding of your changes.
- Merge these changes to `main` before continuing.

🔗 [See all ConsensusVersion uses](https://github.com/search?q=repo%3Apokt-network%2Fpoktroll+ConsensusVersion&type=code)

---

## 2. Prepare a New Upgrade Plan

:::tip Reference

- Review [Pocket Network's historical.go](https://github.com/pokt-network/poktroll/tree/main/app/upgrades) for past upgrades.
- See [Cosmos SDK upgrade docs](https://docs.cosmos.network/main/build/building-apps/app-upgrade).
  :::

**Checklist:**

1. **Select SHAs**

   - Find the SHA of the last public [release](https://github.com/pokt-network/poktroll/releases/)
   - Find the SHA for the new release (usually `main`)
   - Compare them:

     ```bash
     https://github.com/pokt-network/poktroll/compare/v<LAST_RELEASE>..<YOUR_SHA>
     ```

2. **Identify Breaking Changes**

   - Manually inspect the diff for parameter/authorization/state changes

3. **Update Upgrade Plan**
   - Edit `app/upgrades.go` and add your upgrade to `allUpgrades`
   - If you change protobufs, see [protobuf deprecation](./5_protobuf_upgrades.md)
   - Example PR: [#1202](https://github.com/pokt-network/poktroll/pull/1202/files)

**⚠️DO NOT PROCEED until these changes are merged⚠️**

---

## 3. Create a GitHub Release

:::note
See [all releases](https://github.com/pokt-network/poktroll/releases).
:::

1. **Tag the release:**

   - Use one of:

     ```bash
     make release_tag_bug_fix
     # or
     make release_tag_minor_release
     ```

   - Follow on-screen prompts.

2. **Publish the release:**

   - [Draft a new release](https://github.com/pokt-network/poktroll/releases/new)
   - Use the tag from above.

3. **Document the release:**

   - Click `Generate release notes` in the GitHub UI.
   - Add this section **ABOVE** the auto-generated notes:

     ```markdown
     ## Protocol Upgrades

     | Category                     | Applicable | Notes                                                                                  |
     | ---------------------------- | ---------- | -------------------------------------------------------------------------------------- |
     | Planned Upgrade              | ✅         | New features.                                                                          |
     | Consensus Breaking Change    | ✅         | Yes, see upgrade here: https://github.com/pokt-network/poktroll/tree/main/app/upgrades |
     | Manual Intervention Required | ❌         | Cosmosvisor managed everything well .                                                  |
     | Upgrade Height               | ❓         | TBD                                                                                    |

     **Legend**:

     - ✅ - Yes
     - ❌ - No
     - ❓ - Unknown/To Be Determined
     - ⚠️ - Warning/Caution Required

     ## What's Changed

     <!-- Auto-generated GitHub Release Notes continue here -->
     ```

   - Use ❓ and **TBD** for unknowns; fill these in after testing.

4. **Set as a pre-release** (change to `latest release` after upgrade completes).

---

## 4. Write an Upgrade Transaction (JSON file)

:::tip
See [v0.1.2 upgrade transactions](https://github.com/pokt-network/poktroll/pull/1204) for examples.
:::

**How to generate:**

```bash
./tools/scripts/upgrades/prepare_upgrade_tx.sh v<YOUR_VERSION>.<YOUR_RELEASE>.<YOUR_PATCH>
```

Example:

```bash
./tools/scripts/upgrades/prepare_upgrade_tx.sh v0.1.2
```

This will create:

```bash
tools/scripts/upgrades/upgrade_tx_vX.Y.Z_alpha.json
tools/scripts/upgrades/upgrade_tx_vX.Y.Z_beta.json
tools/scripts/upgrades/upgrade_tx_vX.Y.Z_local.json
tools/scripts/upgrades/upgrade_tx_vX.Y.Z_main.json
```

:::info
You will need to update the `height` before submitting each one. _More on this later..._
:::

**Example JSON snippet:**

```json
{
  "body": {
    "messages": [
      {
        "@type": "/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade",
        "authority": "pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t",
        "plan": {
          "name": "v0.0.4",
          "height": "30",
          "info": "{\"binaries\":{...}}"
        }
      }
    ]
  }
}
```

---

## 5. Validate the Upgrade Binary URLs (Live Network Only)

**⚠️Critical: The binary URLs and checksum must be correct, or Cosmovisor will fail⚠️**

Install `go-getter` if you don't have it:

```bash
go install github.com/hashicorp/go-getter/cmd/go-getter@latest
```

And check all binary URLs:

```bash
jq -r '.body.messages[0].plan.info | fromjson | .binaries[]' $PATH_TO_UPGRADE_TRANSACTION_JSON | while IFS= read -r url; do
  go-getter "$url" .
done
```

Expected output:

```bash
success!
success!
success!
success!
```

**⚠️DO NOT PROCEED until all URLs validate⚠️**

---

## 6. Test the New Release

- Follow [Testing Protocol Upgrades](./3_testing_upgrades.md) **before** submitting any transactions.
- If you find an issue, you'll need to repeat the steps as needed (update plan, release, transactions, etc.).

---

## 7. Submit the Upgrade Onchain (Alpha)

This step is parameterized so you can use it for any network (Alpha, Beta, or MainNet). Substitute the variables below as needed.

**Variables:**

- `NETWORK`: one of `pocket-alpha`, `pocket-beta`, or `pocket`
- `RPC_ENDPOINT`: The RPC endpoint for the network (e.g., `https://shannon-testnet-grove-rpc.alpha.poktroll.com`)
- `UPGRADE_TX_JSON`: Path to the upgrade transaction JSON (e.g., `tools/scripts/upgrades/upgrade_tx_v0.1.2_alpha.json`)
- `FROM_ACCOUNT`: The account submitting the transaction (e.g., `pnf_alpha`)
- `TX_HASH`: The hash of the submitted transaction (for monitoring)

**Step-by-step:**

1. Get the RPC endpoint for `<NETWORK>`. Example for Alpha [here](https://dev.poktroll.com/tools/tools/shannon_alpha).
2. Update the `height` in your upgrade transaction JSON:

   ```bash
   # Get the current height
   CURRENT_HEIGHT=$(pocketd status --node <RPC_ENDPOINT> | jq '.sync_info.latest_block_height' | tr -d '"')
   # Add 5 blocks (arbitrary, adjust as needed)
   UPGRADE_HEIGHT=$((CURRENT_HEIGHT + 5))
   # Update the JSON
   sed -i.bak "s/\"height\": \"[^\"]*\"/\"height\": \"$UPGRADE_HEIGHT\"/" <UPGRADE_TX_JSON>
   ```

3. Submit the transaction:

   ```bash
   pocketd \
     --keyring-backend="test" --home="~/.pocket" \
     --fees=300upokt --chain-id="<NETWORK>" --node <RPC_ENDPOINT> \
     tx authz exec <UPGRADE_TX_JSON> --from=<FROM_ACCOUNT>
   ```

   :::tip Grove Employees 🌿

   If you're a Grove Employee, you can use the helpers [here](https://www.notion.so/buildwithgrove/Playbook-Streamlining-rc-helpers-for-Shannon-Alpha-Beta-Main-Network-Environments-152a36edfff680019314d468fad88864?pvs=4) to use this wrapper:

   ```bash
   pkd_<NETWORK>_tx authz exec <UPGRADE_TX_JSON> --from=<FROM_ACCOUNT>
   ```

   :::

4. Verify the upgrade is planned onchain:

   ```bash
   pocketd query upgrade plan --node <RPC_ENDPOINT>
   ```

5. (Optional) Watch the transaction (using the TX_HASH from step 3):

   ```bash
   watch -n 5 "pocketd query tx --type=hash <TX_HASH> --node <RPC_ENDPOINT>"
   ```

6. After upgrade, verify node version aligns with what's in `<UPGRADE_TX_JSON>`:

   ```bash
   curl -s <RPC_ENDPOINT>/abci_info | jq '.result.response.version'
   ```

:::tip Grove Employees 🌿

- Use logging/observability tools to monitor full nodes & validators.
- Only proceed to Beta/MainNet after Alpha is successful.
- [Connect to our cluster](https://www.notion.so/buildwithgrove/Playbook-Connecting-to-Vultr-Protocol-k8s-cluster-protocol-nj-162a36edfff680608c30ff9eebd3e605?pvs=4) to inspect logs and pod status

:::

**⚠️ DO NOT PROCEED until the changes from step (2) are merged assuming the upgrade suceeded⚠️**

---

## 8. Update the `homebrew-tap` Formula

Once the upgrade is validated, update the tap so users can install the new CLI.

**Steps:**

```bash
git clone git@github.com:pokt-network/homebrew-pocketd.git
cd homebrew-pocketd
make tap_update_version
git commit -am "Update pocket tap from v.X1.Y1.Z1 to v.X1.Y2.Z2"
git push
```

**Reinstall the CLI:**

```bash
brew reinstall pocketd
```

**Or install for the first time:**

```bash
brew tap pocket-network/homebrew-pocketd
brew install pocketd
```

See [pocketd CLI docs](../../tools/user_guide/pocketd_cli.md) for more info.

---

## 9. Submit the Upgrade on Beta & MainNet

Repeat [Step 7: Submit the Upgrade Onchain](#7-submit-the-upgrade-onchain-alpha) with the appropriate parameters for Beta and MainNet:

- Use the correct `<RPC_ENDPOINT>` for Beta or MainNet
- Use the correct `<UPGRADE_TX_JSON>` (e.g., `upgrade_tx_v0.1.2_beta.json` or `upgrade_tx_v0.1.2_main.json`)
- Use the correct sender account for each network

This ensures a single, copy-pasta-friendly process for all networks.

**⚠️ DO NOT PROCEED until the changes from step (2) are merged assuming the upgrade succeeded⚠️**

---

## 10. Troubleshooting & Canceling an Upgrade

**If you need to cancel, see:**

- [Chain Halt Troubleshooting](./7_chain_halt_troubleshooting.md)
- [Failed upgrade contingency plan](./8_contigency_plans.md)
- [Chain Halt Recovery](./9_recovery_from_chain_halt.md)

**Checklist:**

1. Follow [Protocol Upgrade Procedure](3_testing_upgrades.md)
2. Update the [Upgrade List](./4_upgrade_list.md)
3. Deploy a full node on TestNet and verify sync (see [Full Node Quickstart Guide](../cheat_sheets/full_node_cheatsheet.md))

---

## Before You Finish

- [ ] All steps above are checked off
- [ ] All releases, plans, and transactions are published and tested
- [ ] The upgrade transaction json files with the updated height are merged in
- [ ] You have communicated upgrade details to the team/community
- [ ] You have prepared for rollback/troubleshooting if needed
