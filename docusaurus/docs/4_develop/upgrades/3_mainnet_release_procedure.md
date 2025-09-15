---
title: MainNet Release Procedure
sidebar_position: 3
---

:::note Purpose of this document

Operational (non-technical) instructions on releasing an upgrade to MainNet

:::

## Table of Contents <!-- omit in toc -->

- [1. Protocol Upgrade Preparation](#1-protocol-upgrade-preparation)
- [2 Keep Clear Communication](#2-keep-clear-communication)
- [3. BEFORE the Day of the Upgrade](#3-before-the-day-of-the-upgrade)
  - [3.1 Choose a height](#31-choose-a-height)
  - [3.2 Submit the Upgrade on MainNet](#32-submit-the-upgrade-on-mainnet)
  - [3.3 Broadcast Telegram Announcement](#33-broadcast-telegram-announcement)
- [4. ON the Day of the Upgrade](#4-on-the-day-of-the-upgrade)
  - [4.1 Prepare another snapshot](#41-prepare-another-snapshot)
  - [4.2 Monitor the Upgrade](#42-monitor-the-upgrade)
  - [4.3 Create a post-upgrade announcement](#43-create-a-post-upgrade-announcement)
  - [4.4 Update the GitHub Release Notes](#44-update-the-github-release-notes)
  - [4.5 Update the Documentation Upgrade List](#45-update-the-documentation-upgrade-list)
  - [4.6 Send out an announcement to all exchanges](#46-send-out-an-announcement-to-all-exchanges)
- [5. Update the `pocketd` binary](#5-update-the-pocketd-binary)
- [6. How to Cancel an Upgrade](#6-how-to-cancel-an-upgrade)
  - [Verify Upgrade Status](#verify-upgrade-status)

## 1. Protocol Upgrade Preparation

## 2 Keep Clear Communication

Keep the following stakeholders in the loop along the way

1. Pocket Network Discord Server; [Beta TestNet](https://discord.com/channels/553741558869131266/1384591252758200330) and [MainNet](https://discord.com/channels/553741558869131266/1234943674903953529)
2. Grove [Pocketd](https://discord.com/channels/824324475256438814/1138895490331705354) Discord
3. Exchanges that support Pocket Network and communicate [via telegram](https://github.com/pokt-network/poktroll/blob/main/.github/workflows/telegram-send-message.yml)

The format of the announcements is always changing so you can reference prior ones below:

- [Beta TestNet Announcement Channel](https://discord.com/channels/553741558869131266/1384589692355477696)
- [MainNet Announcement Channel](https://discord.com/channels/553741558869131266/1384589604153331732)

## 3. BEFORE the Day of the Upgrade

We'll use `v0.1.29` as an example for this section.

### 3.1 Choose a height

1. Visit the [MainNet Grafana Dashboard](https://grafana.poktroll.com/goto/5XmC4RjNR?orgId=1) to get the current height of the blockchain
2. Review the latest block times of the network by checking network stats, [grove's infra](https://github.com/buildwithgrove/infrastructure/blob/dfbc02c57bbc5e61ae860393ec35d45b6a6fc3d5/environments/protocol/vultr-sgp/kubernetes-manifests/mainnet/config-files.yaml#L505) or [config.toml](https://github.com/pokt-network/pocket-network-genesis/blob/master/shannon/mainnet/config.toml); _usually 30s per block_.
3. Determine a future height that gives the ecosystem a few days to prepare.
4. For your particular upgrade (e.g. `v0.1.29`), update the `height` in `tools/scripts/upgrades/upgrade_tx_v0.1.29_main.json`:

### 3.2 Submit the Upgrade on MainNet

Run the following command and follow the instructions:

```bash
./tools/scripts/upgrades/submit_upgrade.sh main v0.1.29 --instruction-only
```

You should end up running a command similar to the following:

```bash
pocketd \
    --keyring-backend="test" --home="~/.pocket" \
    --fees=300upokt --network=main  --from=pokt18808wvw0h4t450t06uvauny8lvscsxjfyua7vh \
    tx authz exec tools/scripts/upgrades/upgrade_tx_v0.1.29_main.json
```

And you can verify it is onchain like so:

```bash
pocketd query upgrade plan --network=main -o json | jq
```

### 3.3 Broadcast Telegram Announcement

Firstly, install the [gh CLI](https://cli.github.com/)

Prepare the announcement like so (using a concrete example for `v0.1.29`)

```bash
cat <<'EOF' >> release_prep_announcement.txt
📢 Pocket Network Upgrade Notice 📢

v0.1.29 is scheduled to go live approximately 10:00 PST on Tuesday (09/16/2025) at block height 382,991.

Find all the details here: https://github.com/pokt-network/poktroll/releases/tag/v0.1.29.

EOF
```

Then, run a test broadcast:

```bash
make telegram_test_broadcast_msg MSG_FILE=release_prep_announcement.txt
```

If it looks good, broadcast it to all exchanges:

```bash
make telegram_broadcast_msg MSG_FILE=release_prep_announcement.txt
```

## 4. ON the Day of the Upgrade

### 4.1 Prepare another snapshot

See the instruction in [Protocol Upgrade Preparation](2_upgrade_preparation.md) on how to prepare a snapshot.

You can find existing snapshots at [snapshots.us-nj.poktroll.com](https://snapshots.us-nj.poktroll.com).

### 4.2 Monitor the Upgrade

Run the following command for your upgrade version and use the recommended commands
and dashboards to monitor the upgrade:

```bash
./tools/scripts/upgrades/submit_upgrade.sh main v0.1.29 --instruction-only
```

:::warning

Wait for the upgrade to complete before proceeding to the next step.

:::

### 4.3 Create a post-upgrade announcement

See the instruction in [Protocol Upgrade Preparation](2_upgrade_preparation.md) to create a post-upgrade snapshot.

### 4.4 Update the GitHub Release Notes

Generate a table of the upgrade heights and tx hashes like so:

```bash
./tools/scripts/upgrades/prepare_upgrade_release_notes.sh v0.1.29
```

Insert the table above the auto-generated [release notes](https://github.com/pokt-network/poktroll/releases).

👉 **Mark it as `latest release`** 👈

### 4.5 Update the Documentation Upgrade List

Update the [Upgrade List Documentation](6_upgrade_list.md) with the new upgrade.

Use the [release notes](https://github.com/pokt-network/poktroll/releases/latest) to populate the upgrade list.

### 4.6 Send out an announcement to all exchanges

Assuming you have the [gh cli](https://cli.github.com/) installed, run the following command and verify the message is properly formatted:

```bash
make telegram_test_release
```

You can then broadcast the release to all exchanges by running:

```bash
make telegram_release_notify
```

:::warning TODO: Releases that are too long

You might get an error that the [message is too long](https://github.com/pokt-network/poktroll/actions/runs/15860176445/job/44715185450).

If this happens, then:

1. Remove unnecessary content from the release notes
2. Run the workflow again
3. Revert the release with all the details

:::

## 5. Update the `pocketd` binary

Once the upgrade is validated, update the tap so users can install the new CLI.

**Run the following steps:**

```bash
git clone git@github.com:pokt-network/homebrew-pocketd.git
cd homebrew-pocketd
make tap_update_version
git commit -am "Update pocket tap from v.<Previous Version> to v.<New Version>"
git push
```

_Note: Make sure to update `v0.1.20` and `v0.1.29` in the commit message above._

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

## 6. How to Cancel an Upgrade

In emergency situations, you may need to cancel a pending upgrade. Use the upgrade script to get the proper cancellation command:

```bash
./tools/scripts/upgrades/submit_upgrade.sh main v0.1.29 --instruction-only
```

This will display the cancellation command at the bottom of the output. You can also run the cancellation command directly:

```bash
pocketd \
    --keyring-backend="test" --home="~/.pocket" \
    --fees=300upokt --network=main \
    tx authz exec tools/scripts/upgrades/cancel_upgrade_main.json --from=pokt18808wvw0h4t450t06uvauny8lvscsxjfyua7vh
```

### Verify Upgrade Status

You can check the current upgrade plan status (whether pending or cancelled) using:

```bash
pocketd query upgrade plan --network=main -o json | jq
```

:::warning Emergency Use Only

The upgrade cancellation command should only be used in emergency situations where the upgrade needs to be stopped before it executes.

:::
