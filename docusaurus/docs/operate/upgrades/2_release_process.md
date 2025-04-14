---
title: Release Process
sidebar_position: 2
---

:::info
This document is intended for core protocol developers.
:::

## Release Process <!-- omit in toc -->

- [1. Determine if the Release is Consensus-Breaking](#1-determine-if-the-release-is-consensus-breaking)
- [2. Create a GitHub Release](#2-create-a-github-release)
- [3. Update the `homebrew-tap` formula](#3-update-the-homebrew-tap-formula)
- [4. Write an Upgrade Plan](#4-write-an-upgrade-plan)
- [5. Issue Upgrade on TestNet](#5-issue-upgrade-on-testnet)
- [6. Issue Upgrade on MainNet](#6-issue-upgrade-on-mainnet)

### 1. Determine if the Release is Consensus-Breaking

Determining if a release is consensus-breaking and documenting it is a 3 step process:

1. **Find consensus breaking changes**: Review merged [Pull Requests (PRs) with the `consensus-breaking` label](https://github.com/pokt-network/poktroll/issues?q=label%3Aconsensus-breaking+) since the last release. It is not a source of truth, but directionality correct.
2. **Update Upgrade List**: If the new release includes an upgrade transaction for automatic upgrades, add the new release to the table in the [Upgrades List](./4_upgrade_list.md).
3. **Verify a Full Node**: Deploy a Full Node on TestNet and allow it to sync and operate for a few days to verify that no accidentally introduced `consensus-breaking` changes affect the ability to sync. See the instructions in the [Quickstart Guide](../../operate/cheat_sheets/full_node_cheatsheet.md) for deploying a Full Node.

:::danger DO NOT SKIP ME

**UPDATE THE INFORMATION IN THE [UPGRADES LIST](./4_upgrade_list.md) DURING THE FOLLOWING STEPS IF ANYTHING CHANGES.**

If we plan to schedule an upgrade at a specific height, update the height.

If the upgrade becomes consensus-breaking, ensure the table remains up-to-date.
:::

### 2. Create a GitHub Release

:::tip GitHub Releases

You can find all existing releases [here](https://github.com/pokt-network/poktroll/releases).

:::

Creating a GitHub release is a 3 step process:

1. **Tag the release**: Create a new tag using either `make release_tag_bug_fix` or `make release_tag_minor_release` commands.
2. **Publish the release**: Create a new release in GitHub using the [Draft a new release button](https://github.com/pokt-network/poktroll/releases/new) feature.
3. **Document the release**: Append and complete the following section above the auto-generated GitHub release notes:

   ```markdown
   ## Protocol Upgrades

   <!--
   IMPORTANT:If this release will be used to issue upgrade on the network, add a link to the upgrade code
   such as https://github.com/pokt-network/poktroll/blob/main/app/upgrades/historical.go#L51.
   -->

   | Category                     | Applicable | Notes                                                                                                                                                                                                                                 |
   | ---------------------------- | ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
   | Planned Upgrade              | ❌         | Not applicable for this release.                                                                                                                                                                                                      |
   | Breaking Change              | ❌         | Not applicable for this release.                                                                                                                                                                                                      |
   | Manual Intervention Required | ✅         | Yes, but only for Alpha TestNet participants. [Follow instructions here](https://dev.poktroll.com/operate/quickstart/docker_compose_walkthrough#restarting-a-full-node-after-re-genesis-) to restart your full node after re-genesis. |
   | Upgrade Height               | ❌         | Not applicable for this release.                                                                                                                                                                                                      |

   **Legend**:

   - ✅ - Yes
   - ❌ - No
   - ❓ - Unknown/To Be Determined
   - ⚠️ - Warning/Caution Required

   ## What's Changed

   <!-- Auto-generated GitHub Release Notes continue here -->
   ```

### 3. Update the `homebrew-tap` formula

```bash
git clone git@github.com:pokt-network/homebrew-pocketd.git
cd homebrew-pocket
make tap_update_version
git commit -am "Update pocket tap from v.X1.Y1.Z1 to vX1.Y2.Z2
git push
```

See the [pocketd CLI docs](../../tools/user_guide/pocketd_cli.md) for more information.

### 4. Write an Upgrade Plan

Protocol upgrades are only necessary for `consensus-breaking` changes. However, we can still issue an upgrade transaction to require Full Nodes and Validators to use a new version.

You can use the following template as a starting point.

```bash
- [ ] Determine the block height at which the upgrade should occur.
  - Selected height: `INSERT_BLOCK_HEIGHT`
- [ ] Update the information in the [Upgrades List](4_upgrade_list.md) and the GitHub Release.
  - Upgrade details: `INSERT_LINK_TO_UPGRADE`
- [ ] Inform the community about the planned upgrade.
  - Announcement: `INSERT_LINK_TO_ANNOUNCEMENT`
- [ ] Prepare a contingency plan to address potential issues.
```

### 5. Issue Upgrade on TestNet

- Follow the [Upgrade Procedure](3_upgrade_procedure.md) to upgrade existing/running Full Nodes and Validators to the new version of `pocket`.
- Monitor the network's health metrics to identify any significant changes, such as the loss of many validators due to an unexpected consensus-breaking change.

### 6. Issue Upgrade on MainNet

- Repeat the upgrade process on the MainNet, following the same steps as on the TestNet.
- Ensure that the upgrade height is set correctly and communicated to the community.
- Monitor the network closely during and after the upgrade to ensure a smooth transition.

:::note

TODO_IMPROVE(@olshansk, @okdas): Link to real notion docs after we've iterated on this process a few times:

:::
