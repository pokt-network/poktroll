---
title: Release Process
sidebar_position: 4
---

## Release Process <!-- omit in toc -->

- [1. Determine if the Release is Consensus-Breaking](#1-determine-if-the-release-is-consensus-breaking)
- [2. Create a GitHub Release](#2-create-a-github-release)
  - [Legend](#legend)
- [3. Write an Upgrade Plan](#3-write-an-upgrade-plan)
- [4. Issue Upgrade on TestNet](#4-issue-upgrade-on-testnet)
- [5. Issue Upgrade on MainNet](#5-issue-upgrade-on-mainnet)

:::info
This document is for the Pocket Network protocol team's internal use only.
:::

### 1. Determine if the Release is Consensus-Breaking

:::note

TODO(#791): The process of adding the `consensus-breaking` label is still not foolproof.

:::

- **Find consensus breaking changes**: Review merged Pull Requests (PRs) with the `consensus-breaking` label.
  If any exist, assume the release will require an upgrade.
  [Here is a link](https://github.com/pokt-network/poktroll/pulls?q=sort%3Aupdated-desc+is%3Apr+is%3Amerged+label%3Aconsensus-breaking) for convenience.

- **Verify a Full Node**: Deploy a Full Node on TestNet and allow it to sync and operate for a few days to verify that no accidentally introduced consensus-breaking changes affect the ability to sync. See the instructions in the [Quickstart Guide](../../operate/quickstart/docker_compose_debian_cheatsheet.md) for deploying a Full Node.

- **Update Upgrade List**: If the new release includes an upgrade transaction for automatic upgrades, add the new release to the table in the [Upgrades List](./upgrade_list.md).

:::danger

**UPDATE THE INFORMATION IN THE [UPGRADES LIST](./upgrade_list.md) DURING THE FOLLOWING STEPS IF ANYTHING CHANGES.** If we plan to schedule an upgrade at a specific height, update the height. If the upgrade becomes consensus-breaking, ensure the table remains up-to-date.
:::

### 2. Create a GitHub Release

:::tip

You can find an example [here](https://github.com/pokt-network/poktroll/releases/tag/v0.0.7).

:::

- **Tag the release**: Create a new tag using the `make release_tag_bug_fix` or `make release_tag_minor_release` commands.
- **Publish the release**: Create a new release in GitHub, using the "Generate release notes" button in the GitHub UI.
- **Document the release**: Append and complete the following section above the generated GitHub release notes:

```text
## Protocol Upgrades

<!--
IMPORTANT:If this release will be used to issue upgrade on the network, add a link to the upgrade code
such as https://github.com/pokt-network/poktroll/blob/main/app/upgrades/historical.go#L51.
-->

- **Planned Upgrade:** ❌ Not applicable for this release.
- **Breaking Change:** ❌ Not applicable for this release.
- **Manual Intervention Required:** ✅ Yes, but only for Alpha TestNet participants. If you are participating, please follow the [instructions provided here](https://dev.poktroll.com/operate/quickstart/docker_compose_walkthrough#restarting-a-full-node-after-re-genesis-) for restarting your full node after re-genesis.
- **Upgrade Height:** ❌ Not applicable for this release.

## What's Changed

<!-- GitHub Release Notes continue here -->
```

#### Legend

- ✅ - Yes
- ❌ - No
- ❓ - Unknown/To Be Determined
- ⚠️ - Warning/Caution Required

### 3. Write an Upgrade Plan

Protocol upgrades are only necessary for `consensus-breaking` changes. However, we can still issue an upgrade transaction to require Full Nodes and Validators to use a new version.

You can use the following template as a starting point.

```bash
- [ ] Determine the block height at which the upgrade should occur.
  - Selected height: `INSERT_BLOCK_HEIGHT`
- [ ] Update the information in the [Upgrades List](./upgrade_list.md) and the GitHub Release.
  - Upgrade details: `INSERT_LINK_TO_UPGRADE`
- [ ] Inform the community about the planned upgrade.
  - Announcement: `INSERT_LINK_TO_ANNOUNCEMENT`
- [ ] Prepare a contingency plan to address potential issues.
```

### 4. Issue Upgrade on TestNet

- Follow the [Upgrade Procedure](./upgrade_procedure.md) to upgrade existing/running Full Nodes and Validators to the new version of `poktroll`.
- Monitor the network's health metrics to identify any significant changes, such as the loss of many validators due to an unexpected consensus-breaking change.

### 5. Issue Upgrade on MainNet

- Repeat the upgrade process on the MainNet, following the same steps as on the TestNet.
- Ensure that the upgrade height is set correctly and communicated to the community.
- Monitor the network closely during and after the upgrade to ensure a smooth transition.

:::note

TODO_IMPROVE(@olshansk, @okdas): Link to real notion docs after we've iterated on this process a few times:

:::
