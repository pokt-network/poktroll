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
- [4. Follow the Protocol Upgrade Procedure](#4-follow-the-protocol-upgrade-procedure)
- [5. Issue the Upgrade](#5-issue-the-upgrade)

### 1. Determine if the Release is Consensus-Breaking

A protocol upgrade is only necessary if there are `consensus-breaking` changes.

A release can still be made without `consensus-breaking` changes, but it will not require a protocol upgrade.

**Identify consensus breaking changes** by:

1. Reviewing merged [Pull Requests (PRs) with the `consensus-breaking` label](https://github.com/pokt-network/poktroll/issues?q=label%3Aconsensus-breaking+) since the last release. It is not a source of truth, but directionality correct.
2. Looking for breaking changes in `.proto` files
3. Looking for breaking changes in the `x/` directories
4. Identifying new onchain parameters or authorizations

:::info Non-exhaustive list

Note that the above is a non-exhaustive list and requires protocol expertise to identify all potential `consensus-breaking` changes.
:::

### 2. Create a GitHub Release

:::tip GitHub Releases

You can find all existing releases [here](https://github.com/pokt-network/poktroll/releases).

:::

Creating a GitHub release is a 3 step process:

1. **Tag the release**: Create a new tag using either `make release_tag_bug_fix` or `make release_tag_minor_release` commands.
2. **Publish the release**: Create a new release in GitHub using the [Draft a new release button](https://github.com/pokt-network/poktroll/releases/new) feature.
3. **Document the release**: Append and complete the following section above the auto-generated GitHub release notes. For example:

   ```markdown
   ## Protocol Upgrades

   | Category                     | Applicable | Notes                                                                                  |
   | ---------------------------- | ---------- | -------------------------------------------------------------------------------------- |
   | Planned Upgrade              | ✅         | New features.                                                                          |
   | Consensus Breaking Change    | ✅         | Yes, see upgrade here: https://github.com/pokt-network/poktroll/tree/main/app/upgrades |
   | Manual Intervention Required | ❌         | Cosmosvisor managed everything well .                                                  |
   | Upgrade Height               | ✅         | Planned upgrade height at 69420 (update with actual height once complete) release.     |

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
git commit -am "Update pocket tap from v.X1.Y1.Z1 to v.X1.Y2.Z2
git push
```

See the [pocketd CLI docs](../../tools/user_guide/pocketd_cli.md) for more information.

### 4. Follow the Protocol Upgrade Procedure

:::info

_tl;dr Follow the protocol upgrade procedure in both cases_

:::

If a release is `consensus-breaking`, you'll need to:

1. Follow the instructions in [**Protocol Upgrade Procedure**](./3_upgrade_procedure.md)
2. If applicable, review [**Protobuf Upgrade Procedure**](./5_protobuf_upgrades.md)
3. Update the [**Upgrade List**](./4_upgrade_list.md)
4. **Deploy a Full Node on TestNet** and allow it to sync and operate for a few days to verify that no accidentally introduced `consensus-breaking` changes affect the ability to sync; [Full Node Quickstart Guide](../../operate/cheat_sheets/full_node_cheatsheet.md).

If a release is not `consensus-breaking`, it is still recommended to issue an upgrade transaction in order to:

1. Require Full Nodes and Validators to use a new version of the software
2. Increase visibility of the software running on the network

### 5. Issue the Upgrade

The [Upgrade Procedure](./3_upgrade_procedure.md) should be tested and verified on:

1. LocalNet
2. Alpha TestNet
3. Beta TestNet
4. MainNet

At each step along the way:

- Monitor the network's health metrics to identify any significant changes
- Communicate upgrades heights and status updates with the community
