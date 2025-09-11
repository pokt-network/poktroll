---
title: MainNet Release Procedure
sidebar_position: 3
---

:::note Purpose of this document

Operational (non-technical) instructions on releasing an upgrade to MainNet

:::

## Table of Contents <!-- omit in toc -->

- [1. Protocol Upgrade Preparation](#1-protocol-upgrade-preparation)
- [2. Keep Clear Communication](#2-keep-clear-communication)
- [2. Choose a height](#2-choose-a-height)
- [3. Submit the Upgrade on MainNet](#3-submit-the-upgrade-on-mainnet)
- [4. Broadcast Telegram Announcement](#4-broadcast-telegram-announcement)
- [5. Ensure Snapshots Available](#5-ensure-snapshots-available)
- [6. Update the GitHub Release Notes](#6-update-the-github-release-notes)
- [7. Update the Documentation Upgrade List](#7-update-the-documentation-upgrade-list)
- [9. Day of Upgrade](#9-day-of-upgrade)
  - [9.1 Prepare Another Snapshot Before the Upgrade](#91-prepare-another-snapshot-before-the-upgrade)
  - [9.2 Monitor the Upgrade](#92-monitor-the-upgrade)
  - [9.3 Community Discord Server](#93-community-discord-server)
  - [8.2 Update the GitHub Release](#82-update-the-github-release)
  - [8.3 Telegram Release Bot](#83-telegram-release-bot)
- [8. Update the `pocketd` binary](#8-update-the-pocketd-binary)

## 1. Protocol Upgrade Preparation

Only follow these instructions after you have completed a successful upgrade on Alpha & Beta TestNet
by following the instructions in [Protocol Upgrade Preparation](2_upgrade_preparation.md).

## 2. Keep Clear Communication

Keep the following stakeholders in the loop along the way

1. Pocket Network Discord Server; [Beta TestNet](https://discord.com/channels/553741558869131266/1384591252758200330) and [MainNet](https://discord.com/channels/553741558869131266/1234943674903953529)
2. Grove [Pocketd](https://discord.com/channels/824324475256438814/1138895490331705354) Discord
3. Exchanges that support Pocket Network and communicate [via telegram](https://github.com/pokt-network/poktroll/blob/main/.github/workflows/telegram-send-message.yml)

## 2. Choose a height

1. Visit the [MainNet Grafana Dashboard](https://grafana.poktroll.com/goto/5XmC4RjNR?orgId=1) to get the current height of the blockchain
2. Review the latest block times of the network by checking network stats, [grove's infra](https://github.com/buildwithgrove/infrastructure/blob/dfbc02c57bbc5e61ae860393ec35d45b6a6fc3d5/environments/protocol/vultr-sgp/kubernetes-manifests/mainnet/config-files.yaml#L505) or [config.toml](https://github.com/pokt-network/pocket-network-genesis/blob/master/shannon/mainnet/config.toml); _usually 30s per block_.
3. Determine a future height that gives the ecosystem a few days to prepare.
4. For your particular upgrade (e.g. `v0.1.21`), update the `height` in `tools/scripts/upgrades/upgrade_tx_v0.1.21_main.json`:

## 3. Submit the Upgrade on MainNet

```bash
./tools/scripts/upgrades/submit_upgrade.sh main v0.1.21
```

## 4. Broadcast Telegram Announcement

1. Install the [gh CLI](https://cli.github.com/)
2. Prepare all the exchanges that support Pocket Network and communicate [via telegram](https://github.com/pokt-network/poktroll/blob/main/.github/workflows/telegram-send-message.yml)
3. Run the following command:

```bash
make telegram_broadcast MSG="📣 Update from Pocket Network: `v0.1.21` is scheduled for release in approximately 1-3 days 📣"
```

## 5. Ensure Snapshots Available

## 6. Update the GitHub Release Notes

1. Generate a table of the upgrade heights and tx hashes like so:

   ```bash
   ./tools/scripts/upgrades/prepare_upgrade_release_notes.sh v0.1.21
   ```

2. Insert the table above the auto-generated [release notes](https://github.com/pokt-network/poktroll/releases).
3. Mark it as `latest release`.

## 7. Update the Documentation Upgrade List

Update the [Upgrade List Documentation](6_upgrade_list.md) with the new upgrade.

Use the [release notes](https://github.com/pokt-network/poktroll/releases/latest) to populate the upgrade list.

## 9. Day of Upgrade

Firstly, it goes without saying, keep comms with the entire ecosystem.

### 9.1 Prepare Another Snapshot Before the Upgrade

### 9.2 Monitor the Upgrade

### 9.3 Community Discord Server

Use the template below as a starting point for your release announcement.

In particular, call out:

- Call to action to the community with the new release
- Major new features or changes
- Thank everyone for their support and whoever was involved

Publish the announcement in the following channels:

- [Discord Beta TestNet Announcement Channel](https://discord.com/channels/553741558869131266/1384589692355477696)
- [Discord MainNet Announcement Channel](https://discord.com/channels/553741558869131266/1384589604153331732)

### 8.2 Update the GitHub Release

Use the template below as a start point for your release announcement. In particular, call out:

- Any other upcoming releases in the near future
- Provide support if they need help running a node

Once you've updated the GitHub release notes, set it as `latest release`.

:::tip v0.1.22 Example

For a full example see the release notes for [v0.1.22](https://github.com/pokt-network/poktroll/releases/tag/v0.1.22).

:::

<details>
<summary>Example Release Template</summary>

Quick Summary

**❓ What changed**: Minor onchain changes to support Morse to Shannon migration and many client changes to improve RelayMiner performance.

💾 The latest snapshot is available [here](https://snapshots.us-nj.poktroll.com).

❗️ If you encounter any issues or bugs, please open up a [GitHub issue](https://github.com/pokt-network/poktroll/issues/new/choose) or just ping the team!

🙏 As always, thank you for your support, cooperation and feedback!

🔜 As of writing this release, `v0.1.23` is planned for sometime early next week.

**Endpoint Reminder** - As a reminder, you can use our public endpoint which is always up to date: https://shannon-grove-rpc.mainnet.poktroll.com.

If you use the Cosmos SDK [cosmovisor](https://docs.cosmos.network/main/build/tooling/cosmovisor), or the [full node script](https://dev.poktroll.com/operate/cheat_sheets/full_node_cheatsheet) built by [Grove](https://www.grove.city/), your nodes should have automatically upgraded.

**🌿 Open Offer from Grove** - As a close partner, we’re also happy to spin up a dedicated endpoint for your exchange specifically. Just let us know if you ever need this in the future!

---

</details>

### 8.3 Telegram Release Bot

If you have the [gh CLI](https://cli.github.com/) installed, you can simply run:

```bash
make telegram_release_notify
```

You can also test the release notification by running:

```bash
make telegram_test_release
```

:::

After setting it as `latest release`, use the [GitHub workflow](https://github.com/pokt-network/poktroll/blob/main/.github/workflows/telegram-notify-release.yml) to automatically notify the Telegram groups.

Go to [this link](https://github.com/pokt-network/poktroll/actions/workflows/telegram-notify-release.yml) and click `Run workflow`.

This will send the details in the GitHub release to all exchanges.

:::warning TODO: Releases that are too long

You might get an error that the [message is too long](https://github.com/pokt-network/poktroll/actions/runs/15860176445/job/44715185450).

If this happens, then:

1. Remove unnecessary content from the release notes
2. Run the workflow again
3. Revert the release with all the details

:::

## 8. Update the `pocketd` binary

Once the upgrade is validated, update the tap so users can install the new CLI.

**Run the following steps:**

```bash
git clone git@github.com:pokt-network/homebrew-pocketd.git
cd homebrew-pocketd
make tap_update_version
git commit -am "Update pocket tap from v.<Previous Version> to v.<New Version>"
git push
```

_Note: Make sure to update `v0.1.20` and `v0.1.21` in the commit message above._

**Reinstall the CLI:**

```bash
brew reinstall pocketd
```

OR

```bash
curl -sSL https://raw.githubusercontent.com/pokt-network/poktroll/main/tools/scripts/pocketd-install.sh | bash -s -- --upgrade
```

**Alternatively, install it for the first time:**

```bash
brew tap pocket-network/homebrew-pocketd
brew install pocketd
```

OR

```bash
curl -sSL https://raw.githubusercontent.com/pokt-network/poktroll/main/tools/scripts/pocketd-install.sh | bash
```
