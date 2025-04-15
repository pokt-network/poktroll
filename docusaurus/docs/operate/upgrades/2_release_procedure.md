---
title: Protocol Upgrade Release Procedure
sidebar_position: 2
---

:::warning Technical expertise required
This document is intended for core protocol developers and may contain complex technical details.

Make sure to read [When is an Protocol Upgrade Warranted?](./1_protocol_upgrades.md#when-is-an-protocol-upgrade-warranted) for more details.
:::

## Table of Contents <!-- omit in toc -->

- [1. Ensure `ConsensusVersion` is updated](#1-ensure-consensusversion-is-updated)
- [2. Prepare a New Upgrade Plan](#2-prepare-a-new-upgrade-plan)
  - [2.1 Testing before merging (for seasoned upgraders only)](#21-testing-before-merging-for-seasoned-upgraders-only)
- [3. Create a GitHub Release](#3-create-a-github-release)
- [4. Write an Upgrade Transaction (json file)](#4-write-an-upgrade-transaction-json-file)
  - [4.1 Validate the Upgrade Binary URLs (live network only)](#41-validate-the-upgrade-binary-urls-live-network-only)
  - [4.2 Release Order of Operations](#42-release-order-of-operations)
- [5. Test the New Release](#5-test-the-new-release)
- [6. Update the `homebrew-tap` formula](#6-update-the-homebrew-tap-formula)
- [7. Submit the Upgrade Onchain](#7-submit-the-upgrade-onchain)
  - [7.1 \[Optional\] Cancel the Upgrade Plan (if needed)](#71-optional-cancel-the-upgrade-plan-if-needed)
- [8. Test \& Champion the Upgrade on All Networks](#8-test--champion-the-upgrade-on-all-networks)

### 1. Ensure `ConsensusVersion` is updated

Ensure the [ConsensusVersion](https://github.com/search?q=repo%3Apokt-network%2Fpoktroll%20ConsensusVersion&type=code) is bumped for all modules with `state-breaking` (i.e. not just `consensus-breaking`) changes.

This will require manual code inspection and understanding of the changes.

⚠️ **Merge in these changes before proceeding.** ⚠️

### 2. Prepare a New Upgrade Plan

:::tip Reference examples

Review all [previous upgrades](https://github.com/pokt-network/poktroll/tree/main/app/upgrades) for reference.

- Refer to `historical.go` for past upgrades and examples.
- Consult the [Cosmos SDK](https://docs.cosmos.network/) documentation on upgrades for additional guidance on [building-apps/app-upgrade](https://docs.cosmos.network/main/build/building-apps/app-upgrade) and [modules/upgrade](https://docs.cosmos.network/main/build/).

:::

1. `sha` selection
   - Identify the `sha` of the last public [release](https://github.com/pokt-network/poktroll/releases/)
   - Choose the `sha` of new release, which will likely be [main](https://github.com/pokt-network/poktroll/commits/main/)
   - Compare the diff between the two shas like so: `https://github.com/pokt-network/poktroll/compare/v<LAST_RELEASE>..<YOUR_SHA>`; [_example_](https://github.com/pokt-network/poktroll/compare/v0.0.11..7541afd6d89a12d61e2c32637b535f24fae20b58)
2. Breaking change identification
   - Between the two `sha`s above, identify any parameter changes, authorizations, functions or other state changes.
   - _This will require manual code inspection and understanding of the changes._
3. Upgrade Plan
   - Update `app/upgrades.go` file to include the new upgrade plan in `allUpgrades`
   - See [this PR](https://github.com/pokt-network/poktroll/pull/1202/files) for an example.
   - If modifying protobuf definitions, reference the approach in [protobuf deprecation](./5_protobuf_upgrades.md) for backward compatibility.

⚠️ **Merge in these changes before proceeding.** ⚠️

#### 2.1 Testing before merging (for seasoned upgraders only)

Changes should be tested before they are merged. When it comes to upgrades, this is even more important but is nuanced and requires experience.

If this is your first time managing an upgrade, we recommend following the instructions
in this document verbatim. You will be prompted to test the upgrade on LocalNet by
following the instructions in [Testing Upgrades](./3_testing_upgrades.md) after you publish the release.

If you are a seasoned protocol upgrader, consider testing the changes first before publishing the release.

### 3. Create a GitHub Release

:::note GitHub Releases

You can find all existing GitHub releases [here](https://github.com/pokt-network/poktroll/releases).

:::

Creating a GitHub release is a 3 step process:

1. **Tag the release**: Create a new tag using either `make release_tag_bug_fix` or `make release_tag_minor_release` commands and following the on screen instructions.
2. **Publish the release**: Create a new release in GitHub using the [Draft a new release button](https://github.com/pokt-network/poktroll/releases/new) feature.
3. **Document the release**: Click `Generate release notes` in the GitHub UI and append the following section above the auto-generated GitHub release notes. For example:

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

4. Use ❓ and `TBD` for unknown values. These will be edited and filled out after
5. Publish the release as `Set as a pre-release`. This will be changed to `latest release` after the upgrade is completed.

### 4. Write an Upgrade Transaction (json file)

:::tip

See the upgrade transactions for `v0.1.2` [here](https://github.com/pokt-network/poktroll/pull/1204) as an example.

It must reference the release URLs and checksums from the [v0.1.2 release](https://github.com/pokt-network/poktroll/releases/tag/v0.1.2) published on GitHub.

:::

An upgrade transaction includes a [Plan](https://github.com/cosmos/cosmos-sdk/blob/0fda53f265de4bcf4be1a13ea9fad450fc2e66d4/x/upgrade/proto/cosmos/upgrade/v1beta1/upgrade.proto#L14) with specific details about the upgrade.

This information helps schedule the upgrade on the network and provides necessary data for automatic upgrades via `Cosmovisor`.

A typical upgrade transaction includes:

- `name`: Name of the upgrade. It should match the `VersionName` of `upgrades.Upgrade`.
- `height`: The height at which an upgrade should be executed and the node will be restarted.
- `info`: Can be empty. **Only needed for live networks where we want cosmovisor to upgrade nodes automatically**.

When `cosmovisor` is configured to automatically download binaries, it will pull the binary from the link provided in
the upgrade object and perform a hash verification (optional).

Here is an example for reference:

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
          "info": "{\"binaries\":{\"linux/amd64\":\"https://github.com/pokt-network/poktroll/releases/download/v0.0.4/poktroll_linux_amd64.tar.gz?checksum=sha256:49d2bcea02702f3dcb082054dc4e7fdd93c89fcd6ff04f2bf50227dacc455638\",\"linux/arm64\":\"https://github.com/pokt-network/poktroll/releases/download/v0.0.4/poktroll_linux_arm64.tar.gz?checksum=sha256:698f3fa8fa577795e330763f1dbb89a8081b552724aa154f5029d16a34baa7d8\",\"darwin/amd64\":\"https://github.com/pokt-network/poktroll/releases/download/v0.0.4/pocket_darwin_amd64.tar.gz?checksum=sha256:5ecb351fb2f1fc06013e328e5c0f245ac5e815c0b82fb6ceed61bc71b18bf8e9\",\"darwin/arm64\":\"https://github.com/pokt-network/poktroll/releases/download/v0.0.4/pocket_darwin_arm64.tar.gz?checksum=sha256:a935ab83cd770880b62d6aded3fc8dd37a30bfd15b30022e473e8387304e1c70\"}}"
        }
      }
    ]
  }
}
```

You can generate the upgrade transaction JSON files for your release with this command:

```bash
./tools/scripts/upgrades/prepare_upgrade_tx.sh v<YOUR_VERSION>.<YOUR_RELEASE>.<YOUR_PATCH>
```

For example, replacing `v<YOUR_VERSION>.<YOUR_RELEASE>.<YOUR_PATCH>` with `vX.Y.Z`, running this:

```bash
./tools/scripts/upgrades/prepare_upgrade_tx.sh v0.1.2
```

Will generate 4 files:

```bash
tools/scripts/upgrades/upgrade_tx_vX.Y.Z_alpha.json
tools/scripts/upgrades/upgrade_tx_vX.Y.Z_beta.json
tools/scripts/upgrades/upgrade_tx_vX.Y.Z_local.json
tools/scripts/upgrades/upgrade_tx_vX.Y.Z_main.json
```

:::info

Note that you'll need update the `height` in each of them independently before submitting each upgrade.

:::

#### 4.1 Validate the Upgrade Binary URLs (live network only)

The URLs of the binaries contain checksums. It is critical to ensure they are correct.
**Otherwise, Cosmovisor won't be able to download the binaries and go through the upgrade.**

The command below (using tools build by the authors of Cosmosvisor) can be used to achieve the above:

```bash
jq -r '.body.messages[0].plan.info | fromjson | .binaries[]' $PATH_TO_UPGRADE_TRANSACTION_JSON | while IFS= read -r url; do
  go-getter "$url" .
done
```

The output should look like this:

```text
2024/09/24 12:40:40 success!
2024/09/24 12:40:42 success!
2024/09/24 12:40:44 success!
2024/09/24 12:40:46 success!
```

:::tip

`go-getter` can be installed using the following command:

```bash
go install github.com/hashicorp/go-getter/cmd/go-getter@latest
```

:::

⚠️ **Merge in these changes before proceeding AND note that this IS NOT part of the release sha.** ⚠️

#### 4.2 Release Order of Operations

We only know the hashes **AFTER** the release has been cut and CI created artifacts for this version.

If you are an experienced protocol upgrader, you should know what to do.

Otherwise, you will need to either:

- Cut multiple releases and update the transactions each time to streamline the process
- Delete and re-release the same tag

:::warning TODO

**Improve the documents for this section so anyone can follow it.**

:::

### 5. Test the New Release

Follow the instructions in [Testing Protocol Upgrades](./3_testing_upgrades.md) before proceeding to the next step.

If an issue is identified in the upgrade plan you prepare in [step 2](#2-prepare-a-new-upgrade-plan), you may need repeat
the steps above including:

1. Update the source code of the upgrade plan
2. Cutting a new release
3. Preparing new upgrade transactions
4. Etc...

### 6. Update the `homebrew-tap` formula

Once you've validated the upgrade, update the `homebrew-tap` formula so all users can easily download the new CLI.

Update the tap:

```bash
git clone git@github.com:pokt-network/homebrew-pocketd.git
cd homebrew-pocket
make tap_update_version
git commit -am "Update pocket tap from v.X1.Y1.Z1 to v.X1.Y2.Z2
git push
```

Reinstall the CLI yourself:

```bash
brew reinstall pocketd
```

Or install it for the first time:

```bash
brew tap pocket-network/homebrew-pocketd
brew install pocketd
```

See the [pocketd CLI docs](../../tools/user_guide/pocketd_cli.md) for more information.

### 7. Submit the Upgrade Onchain

The `MsgSoftwareUpgrade` can be submitted using the following command:

```bash
pocketd tx authz exec $PATH_TO_UPGRADE_TRANSACTION_JSON --from=pnf
```

If the transaction has been accepted, the upgrade plan can be viewed with this command:

```bash
pocketd query upgrade plan
```

#### 7.1 [Optional] Cancel the Upgrade Plan (if needed)

It is possible to cancel the upgrade before the upgrade plan height is reached.

See [Chain Halt Troubleshooting](./7_chain_halt_troubleshooting.md), [Failed upgrade contigency plan](./8_contigency_plans.md) and [Chain Halt Recovery](./9_recovery_from_chain_halt.md) for more details.

To do so, execute the following make target:

1. Follow the instructions in [**Protocol Upgrade Procedure**](3_testing_upgrades.md)
2. Update the [**Upgrade List**](./4_upgrade_list.md)
3. **Deploy a Full Node on TestNet** and allow it to sync and operate for a few days to verify that no accidentally introduced `consensus-breaking` changes affect the ability to sync; [Full Node Quickstart Guide](../cheat_sheets/full_node_cheatsheet.md).

### 8. Test & Champion the Upgrade on All Networks

The [Upgrade Procedure](3_testing_upgrades.md) should be tested and verified on:

1. LocalNet
2. Alpha TestNet
3. Beta TestNet
4. MainNet

At each step along the way:

- Monitor the network's health metrics to identify any significant changes
- Communicate upgrades heights and status updates with the community
