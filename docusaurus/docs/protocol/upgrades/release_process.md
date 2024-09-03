---
title: Release Process
sidebar_position: 4
---

# Release Process <!-- omit in toc -->

:::info
This document is for the Pocket Network protocol team's internal use only.
:::

- [1. Determine if the Release is Consensus-Breaking](#1-determine-if-the-release-is-consensus-breaking)
- [2. Create a GitHub Release](#2-create-a-github-release)
  - [Legend](#legend)
- [3. Write an Upgrade Plan](#3-write-an-upgrade-plan)
- [4. Issue Upgrade on TestNet](#4-issue-upgrade-on-testnet)
- [5. Issue Upgrade on MainNet](#5-issue-upgrade-on-mainnet)

## 1. Determine if the Release is Consensus-Breaking

- Review merged Pull Requests (PRs) with the `consensus-breaking` label. If any exist, assume the release will require an upgrade.
- Deploy a Full Node on TestNet and allow it to sync and operate for a few days to verify that no accidentally introduced consensus-breaking changes affect the ability to sync.
- If the new release includes an upgrade transaction for automatic upgrades, add the new release to the table in the [Upgrades List](./upgrade_list.md).
  - **UPDATE THE INFORMATION IN THIS TABLE DURING THE FOLLOWING STEPS IF ANYTHING CHANGES.** If we plan to schedule an upgrade at a specific height, update the height. If the upgrade becomes consensus-breaking, ensure the table remains up-to-date.

## 2. Create a GitHub Release

- Create a new tag using the `make tag_bug_fix` or `make tag_minor_release` commands.
- Create a new release in GitHub, using the "Generate release notes" button in the GitHub UI.
- Append and complete the following section before the generated GitHub release notes:

```
## Protocol Upgrades

- **Planned Upgrade:** ❌ Not applicable for this release.
- **Breaking Change:** ❌ Not applicable for this release.
- **Manual Intervention Required:** ✅ Yes, but only for Alpha TestNet participants. If you are participating, please follow the [instructions provided here](https://dev.poktroll.com/operate/quickstart/docker_compose_walkthrough#restarting-a-full-node-after-re-genesis-) for restarting your full node after re-genesis.
- **Upgrade Height:** ❌ Not applicable for this release.

## What's Changed
<!-- GitHub Release Notes continue here -->
```

### Legend

✅ - Yes

❌ - No  

❓ - Unknown/To Be Determined

⚠️ - Warning/Caution Required

## 3. Write an Upgrade Plan

Protocol upgrades are only necessary for consensus-breaking changes. However, we can still issue an upgrade transaction to require Full Nodes and Validators to use the new version.

- Determine the block height at which the upgrade should occur.
- Update the information in the [Upgrades List](./upgrade_list.md) and the GitHub Release.
- Inform the community about the planned upgrade.
- Prepare a contingency plan to address potential issues.

## 4. Issue Upgrade on TestNet

- Follow the [Upgrade Procedure](./upgrade_procedure.md) to upgrade existing/running Full Nodes and Validators to the new version of `poktroll`.
- Monitor the network's health metrics to identify any significant changes, such as the loss of many validators due to an unexpected consensus-breaking change.

## 5. Issue Upgrade on MainNet

- Repeat the upgrade process on the MainNet, following the same steps as on the TestNet.
- Ensure that the upgrade height is set correctly and communicated to the community.
- Monitor the network closely during and after the upgrade to ensure a smooth transition.
