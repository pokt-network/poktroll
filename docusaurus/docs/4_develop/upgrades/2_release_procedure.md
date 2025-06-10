---
title: Protocol Upgrade Release Procedure
sidebar_position: 2
---

:::important
This is the step-by-step (almost) 🖨🍝 checklist for core protocol developers to release protocol upgrades.

**❗ DO NOT PROCEED if you are not comfortable with Git, GitHub releases, scripting, etc❗**
:::

## If this is your first time managing an upgrade, learn the following:

- Ensure you know [When is a Protocol Upgrade Needed?](./1_protocol_upgrades.md#when-is-a-protocol-upgrade-needed)
- Ensure you have push access to [pokt-network/poktroll](https://github.com/pokt-network/poktroll)
- Ensure you have the required CLI tools (`git`, `make`, `jq`, `sed`, `curl`, `go`, `brew`, `pocketd`, etc.)
- Understand `state-breaking` vs `consensus-breaking` changes from the [overview](./1_protocol_upgrades.md)
- Be aware of the list of [previous upgrades](https://github.com/pokt-network/poktroll/tree/main/app/upgrades) for reference
- If you implemented the upgrade, familiarize yourself with [how to test changes locally](3_testing_upgrades_locally.md)

## Table of Contents <!-- omit in toc -->

- [If this is your first time managing an upgrade, learn the following:](#if-this-is-your-first-time-managing-an-upgrade-learn-the-following)
- [1. Prepare a New Upgrade Plan](#1-prepare-a-new-upgrade-plan)
- [3. Create a GitHub Release](#3-create-a-github-release)
- [4. Write an Upgrade Transaction (JSON file)](#4-write-an-upgrade-transaction-json-file)
- [5. Validate the Upgrade Binary URLs (Live Network Only)](#5-validate-the-upgrade-binary-urls-live-network-only)
- [6. Test the New Release Locally](#6-test-the-new-release-locally)
- [7. Submit the Upgrade on Alpha TestNet](#7-submit-the-upgrade-on-alpha-testnet)
- [8. Update the `homebrew-tap` Formula](#8-update-the-homebrew-tap-formula)
- [9. Submit the Upgrade on Beta \& MainNet](#9-submit-the-upgrade-on-beta--mainnet)
- [10. Troubleshooting \& Canceling an Upgrade](#10-troubleshooting--canceling-an-upgrade)
- [Before You Finish](#before-you-finish)
- [TODO](#todo)

<details>
<summary>TODO(@olshansk): Ensure `ConsensusVersion` updates is part of the process</summary>

- Bump the `ConsensusVersion` for all modules with `state-breaking` changes.
- This requires manual inspection and understanding of your changes.
- Merge these changes to `main` before continuing.

🔗 [See all ConsensusVersion uses](https://github.com/search?q=repo%3Apokt-network%2Fpoktroll+ConsensusVersion&type=code)

**⚠️DO NOT PROCEED until these changes are merged⚠️**

</details>

## 1. Prepare a New Upgrade Plan

:::tip Reference

- Review [Pocket Network's historical.go](https://github.com/pokt-network/poktroll/tree/main/app/upgrades) for past upgrades.
- Optional Reference: [Cosmos SDK upgrade docs](https://docs.cosmos.network/main/build/building-apps/app-upgrade).
  :::

**TODO_IN_THIS_PR**: Replace this whole thing with the `vNext` approach.
Make this comply with the vNext approach.

1. **Select SHAs**

   - Find the SHA of the last public [release](https://github.com/pokt-network/poktroll/releases/)
   - Find the SHA for the new release (usually `main`)
   - Compare them:

     ```bash
     https://github.com/pokt-network/poktroll/compare/v<LAST_RELEASE_SHA>..<NEW_RELEASE_SHA>
     ```

2. **Identify Breaking Changes**

   - Manually inspect the diff for parameter/authorization/state changes

3. **Update Upgrade Plan**

   - If any protobufs were changed, make sure to review [protobuf deprecation](./5_protobuf_upgrades.md)
   - Edit `app/upgrades.go` and add your upgrade to `allUpgrades`
   - **Examples**:
     - [v0.1.2](https://github.com/pokt-network/poktroll/pull/1202/files) - State-breaking change with a new parameter; _simple upgrade_
     - [v0.1.3](https://github.com/pokt-network/poktroll/pull/1216/files) - Node software update without consensus-breaking changes; _empty upgrade_

**⚠️DO NOT PROCEED until these changes are merged⚠️**

---

## 3. Create a GitHub Release

:::note
You can review [all prior releases here](https://github.com/pokt-network/poktroll/releases).
:::

1. **Tag the release** using one of the following and follow on-screen prompts:

   ```bash
   make release_tag_bug_fix
   # or
   make release_tag_minor_release
   ```

2. **Publish the release** by:

   - [Drafting a new release](https://github.com/pokt-network/poktroll/releases/new)
   - Using the tag from the step above

3. **Update the description in the release** by:

   - Clicking `Generate release notes` in the GitHub UI
   - Add this table **ABOVE** the auto-generated notes (below)

     ```markdown
     ## Protocol Upgrades

     | Category                     | Applicable | Notes                                |
     | ---------------------------- | ---------- | ------------------------------------ |
     | Planned Upgrade              | ✅         | New features.                        |
     | Consensus Breaking Change    | ✅         | Yes, see upgrade here: #1216         |
     | Manual Intervention Required | ❓         | Cosmosvisor managed everything well. |

     | Network       | Upgrade Height | Upgrade Transaction Hash | Notes |
     | ------------- | -------------- | ------------------------ | ----- |
     | Alpha TestNet | ⚪             | ⚪                       | ⚪    |
     | Beta TestNet  | ⚪             | ⚪                       | ⚪    |
     | MainNet       | ⚪             | ⚪                       | ⚪    |

     **Legend**:

     - ⚠️ - Warning/Caution Required
     - ✅ - Yes
     - ❌ - No
     - ⚪ - Will be filled out throughout the release process / To Be Determined
     - ❓ - Unknown / Needs Discussion

     ## What's Changed

     <!-- Auto-generated GitHub Release Notes continue here -->
     ```

     TODO_IN_THIS_PR:

     - Script to auto-generate descriptions
     - Links to where/how the artifacts are build

4. **Set as a pre-release** (change to `latest release` after upgrade completes).

---

## 4. Write an Upgrade Transaction (JSON file)

Add a make target for this.

:::tip
See [v0.1.2 upgrade transactions](https://github.com/pokt-network/poktroll/pull/1204) for examples.
:::

**Generate the new upgrade transaction JSON files like so**:

```bash
./tools/scripts/upgrades/prepare_upgrade_tx.sh v<YOUR_VERSION>.<YOUR_RELEASE>.<YOUR_PATCH>
```

For example, running:

```bash
./tools/scripts/upgrades/prepare_upgrade_tx.sh v0.1.2
```

:::note 😎 Keep Calm and Wait for CI 😅

If you see an error message like this:

```bash
$ ./tools/scripts/upgrades/prepare_upgrade_tx.sh v0.1.11
Downloading checksum file from https://github.com/pokt-network/poktroll/releases/download/v0.1.11/release_checksum...
Error: Failed to download checksum file
```

This means that CI is still building the release artifacts.
You can check the status of CI by looking for a run corresponding to the new release/tag on [the actions page](https://github.com/pokt-network/poktroll/actions).

:::

Will create:

```bash
tools/scripts/upgrades/upgrade_tx_vX.Y.Z_alpha.json
tools/scripts/upgrades/upgrade_tx_vX.Y.Z_beta.json
tools/scripts/upgrades/upgrade_tx_vX.Y.Z_local.json
tools/scripts/upgrades/upgrade_tx_vX.Y.Z_main.json
```

:::info `height` is not populate in the `*.json` files
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

**⚠️The binary URLs and checksum must be correct for LIVE NETWORKS. Otherwise,cosmovisor will fail⚠️**

Install `go-getter` if you don't have it:

```bash
go install github.com/hashicorp/go-getter/cmd/go-getter@latest
```

And check all binary URLs:

```bash
RELEASE_VERSION=<VERSION> # E.g. "v0.1.11"
for file in ./tools/scripts/upgrades/upgrade_tx_${RELEASE_VERSION}*; do
  echo "Processing $file"
  jq -r '.body.messages[0].plan.info | fromjson | .binaries[]' "$file" | while IFS= read -r url; do
    go-getter "$url" .
  done
done
```

Expected output should look like the following:

```bash
2025/04/16 12:11:36 success!
2025/04/16 12:11:40 success!
2025/04/16 12:11:44 success!
2025/04/16 12:11:48 success!
```

**⚠️DO NOT PROCEED until all URLs validate⚠️**

---

## 6. Test the New Release Locally

- Follow [Testing Protocol Upgrades](3_testing_upgrades_locally.md) **before** submitting any transactions.
- If you find an issue, you'll need to repeat the steps as needed (update plan, release, transactions, etc.).

---

## 7. Submit the Upgrade on Alpha TestNet

This step is parameterized so you can use it for any network (Alpha, Beta, or MainNet). Substitute the variables below as needed.

**Variables:**

- `NETWORK`: one of (`local`, `alpha`, `beta`, `main`)
- `RPC_ENDPOINT`: The RPC endpoint for the network (e.g., `https://shannon-testnet-grove-rpc.alpha.poktroll.com`)
- `UPGRADE_TX_JSON`: Path to the upgrade transaction JSON (e.g., `tools/scripts/upgrades/upgrade_tx_v0.1.2_alpha.json`)
- `FROM_ACCOUNT`: The account submitting the transaction (e.g., `pnf_alpha`)
- `TX_HASH`: The hash of the submitted transaction (for monitoring)

**Step-by-Step instructions:**

1. Get the RPC endpoint for `NETWORK`. Example for Alpha [here](https://dev.poktroll.com/tools/tools/shannon_alpha).
2. Update the `height` in your upgrade transaction JSON ():

   :::tip Export `UPGRADE_TX_JSON`, `RPC_ENDPOINT`, `NETWORK`, and `FROM_ACCOUNT`

   ```bash
   export RPC_ENDPOINT=https://shannon-testnet-grove-rpc.alpha.poktroll.com
   export UPGRADE_TX_JSON="tools/scripts/upgrades/upgrade_tx_v0.1.2_alpha.json"
   export NETWORK=alpha
   export FROM_ACCOUNT=pnf_alpha
   ```

   :::

   ```bash
   # Get the current height
   CURRENT_HEIGHT=$(pocketd q block --network=${NETWORK} -o json | tail -n +2 | jq -r '.header.height') # Add 5 blocks (arbitrary, adjust as needed)
   UPGRADE_HEIGHT=$((CURRENT_HEIGHT + 5))
   # Update the JSON
   sed -i.bak "s/\"height\": \"[^\"]*\"/\"height\": \"$UPGRADE_HEIGHT\"/" ${UPGRADE_TX_JSON}
   # Cat the output file
   cat ${UPGRADE_TX_JSON}
   ```

3. Submit the transaction:

   ```bash
   pocketd \
     --keyring-backend="test" --home="~/.pocket" \
     --fees=300upokt --network=${NETWORK} \
     tx authz exec ${UPGRADE_TX_JSON} --from=${FROM_ACCOUNT}
   ```

   :::tip Grove Employee Helpers 🌿

   If you're a Grove Employee, you can use the helpers [here](https://www.notion.so/buildwithgrove/Playbook-Streamlining-rc-helpers-for-Shannon-Alpha-Beta-Main-Network-Environments-152a36edfff680019314d468fad88864?pvs=4) to use this wrapper:

   ```bash
   pkd_<NETWORK>_tx authz exec ${UPGRADE_TX_JSON} --from=${FROM_ACCOUNT}
   ```

4. Verify the upgrade is planned onchain:

   ```bash
   pocketd query upgrade plan --network=${NETWORK}
   ```

5. Watch the transaction (using the TX_HASH from step 3):

   ```bash
   watch -n 5 "pocketd query tx --type=hash ${TX_HASH} --network=${NETWORK}"
   ```

6. Verify node version aligns with what's in `<UPGRADE_TX_JSON>`:

   ```bash
   curl -s ${RPC_ENDPOINT}/abci_info | jq '.result.response.version'
   ```

7. Once the upgrade is complete, make sure to:
   - Record the upgrade `height` and `tx_hash` in the [GitHub Release](https://github.com/pokt-network/poktroll/releases)
   - Commit the `{UPGRADE_TX_JSON}` file with the final height to `main`

:::tip Grove Employees 🌿

- Use logging/observability tools to monitor full nodes & validators.
- Only proceed to Beta/MainNet after Alpha is successful.
- [Connect to our cluster](https://www.notion.so/buildwithgrove/Playbook-Connecting-to-Vultr-Protocol-k8s-cluster-protocol-nj-162a36edfff680608c30ff9eebd3e605?pvs=4) to inspect logs and pod status

:::

**⚠️ DO NOT PROCEED until the changes from step (2) are merged assuming the upgrade succeeded ⚠️**

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

See [pocketd CLI docs](../../2_explore/2_account_management/1_pocketd_cli.md) for more info.

---

## 9. Submit the Upgrade on Beta & MainNet

Repeat [Step 7: Submit the Upgrade Onchain](#7-submit-the-upgrade-on-alpha-testnet) with the appropriate parameters for Beta and MainNet:

- Use the correct `<RPC_ENDPOINT>` for Beta or MainNet
- Use the correct `<NETWORK>`, (`beta` for Beta or `main` for MainNet)
- Use the correct `<UPGRADE_TX_JSON>` (e.g., `upgrade_tx_v0.1.2_beta.json` or `upgrade_tx_v0.1.2_main.json`)
- Use the correct sender account for each network

This ensures a single, copy-pasta-friendly process for all networks.

**⚠️ DO NOT PROCEED until the changes from step (2) are merged assuming the upgrade succeeded ⚠️**

:::tip Grove Employees 🌿

If you're a Grove Employee, you can use the helpers [here](https://www.notion.so/buildwithgrove/Playbook-Streamlining-rc-helpers-for-Shannon-Alpha-Beta-Main-Network-Environments-152a36edfff680019314d468fad88864?pvs=4) to use this wrapper:

See the instructions in [this PR](https://github.com/pokt-network/poktroll/pull/1219) for copy-pasta
commands on how `v0.1.3` was submitted on Alpha & Beta.

:::

---

## 10. Troubleshooting & Canceling an Upgrade

**If you need to cancel, see:**

- [Chain Halt Troubleshooting](./7_chain_halt_troubleshooting.md)
- [Failed upgrade contingency plan](./8_contigency_plans.md)
- [Chain Halt Recovery](./9_recovery_from_chain_halt.md)

**Checklist:**

1. Follow [Protocol Upgrade Procedure](3_testing_upgrades_locally.md)
2. Update the [Upgrade List](./4_upgrade_list.md)
3. Deploy a full node on TestNet and verify sync (see [Full Node Quickstart Guide](../../1_operate/1_cheat_sheets/2_full_node_cheatsheet.md))

---

## Before You Finish

- [ ] All steps above are checked off
- [ ] All releases, plans, and transactions are published and tested
- [ ] The upgrade transaction json files with the updated height are merged in
- [ ] You have communicated upgrade details to the team/community
- [ ] You have prepared for rollback/troubleshooting if needed
- [ ] You have updated the published release with the final upgrade height on each network

## TODO

The following improvements will streamline this process further

- [ ] Concrete examples of PR examples & descriptions along the way
- [ ] Additional helpers (not automation) for some of the commands throughout

TODO_IN_THIS_PR: Remind the reader to make the release the latest at the very end.
TOD_IN_THIS_PR: Add links to observability:

- https://discord.com/channels/824324475256438814/1382058945920630906
- https://github.com/pokt-network/poktroll/pull/1460

oad here: https://grafana.tooling.grove.city/goto/xL03JALNR?orgId=1
Claims & proofs on mainnet https://explorer.pocket.network/pocket-mainnet - to make sure everything works as expected. Last time I discovered lots of tokens spent on relays on poktscan, but it doesn't work right now. So just waiting for a block with eth claims/proofs.
31 claims in this block
