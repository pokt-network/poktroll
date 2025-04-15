---
title: E2E Release Process
sidebar_position: 2
---

:::info
This document is intended for core protocol developers.
:::

## Table of Contents <!-- omit in toc -->

- [1. Identify the `sha` of the new release](#1-identify-the-sha-of-the-new-release)
- [2. Create a GitHub Release](#2-create-a-github-release)
- [3. Update the `homebrew-tap` formula](#3-update-the-homebrew-tap-formula)
- [4. Follow the Protocol Upgrade Procedure](#4-follow-the-protocol-upgrade-procedure)
- [5. Issue the Upgrade on All Networks](#5-issue-the-upgrade-on-all-networks)

### 1. Identify the `sha` of the new release

Identify all changes since the last release by:

1. Identify the `sha` of the public [release](https://github.com/pokt-network/poktroll/releases/).
2. Choose the `sha` of new release, which will likely be [main](https://github.com/pokt-network/poktroll/commits/main/).
3. Compare the diff between the two shas like so: `https://github.com/pokt-network/poktroll/compare/v<LAST_RELEASE>..<YOUR_SHA>`; ([example](https://github.com/pokt-network/poktroll/compare/v0.0.11..7541afd6d89a12d61e2c32637b535f24fae20b58)).
4. Ensure the [ConsensusVersion](https://github.com/search?q=repo%3Apokt-network%2Fpoktroll%20ConsensusVersion&type=code) is bumped for all modules with `state-breaking` (i.e. not just `consensus-breaking`) changes.

:::tip

Read [When is an Protocol Upgrade Warranted?](./1_protocol_upgrades.md#when-is-an-protocol-upgrade-warranted) for more details on `consensus-breaking` changes.

:::

### 2. Create a GitHub Release

:::note GitHub Releases

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

If a release is not `consensus-breaking` but changes to node software were made, you should still issue an upgrade transaction to:

1. Require Full Nodes and Validators to use a new version of the software
2. Increase visibility of the software running on the network

### 5. Issue the Upgrade on All Networks

The [Upgrade Procedure](./3_upgrade_procedure.md) should be tested and verified on:

1. LocalNet
2. Alpha TestNet
3. Beta TestNet
4. MainNet

At each step along the way:

- Monitor the network's health metrics to identify any significant changes
- Communicate upgrades heights and status updates with the community
