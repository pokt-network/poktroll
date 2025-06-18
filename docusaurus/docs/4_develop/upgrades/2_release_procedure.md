---
title: Protocol Upgrade Release Procedure
sidebar_position: 2
---

:::important
This is the step-by-step (almost) 🖨🍝 checklist for core protocol developers to release protocol upgrades.

**❗ DO NOT PROCEED if you are not comfortable with Git, GitHub releases, scripting, etc❗**
:::

## If this is your first time managing an upgrade: <!-- omit in toc -->

- Ensure you understand [When is a Protocol Upgrade Needed?](./1_protocol_upgrades.md#when-is-a-protocol-upgrade-needed)
- Ensure you have push access to [pokt-network/poktroll](https://github.com/pokt-network/poktroll)
- Ensure you have the required CLI tools (`git`, `make`, `jq`, `sed`, `curl`, `go`, `brew`, `pocketd`, etc.)
- Understand `state-breaking` vs `consensus-breaking` changes from the [overview](./1_protocol_upgrades.md)
- Be aware of the list of [previous upgrades](https://github.com/pokt-network/poktroll/tree/main/app/upgrades) for reference
- If you implemented the upgrade, familiarize yourself with [how to test changes locally](3_testing_upgrades_locally.md)

## Table of Contents <!-- omit in toc -->

- [0. Communicate](#0-communicate)
- [1. Prepare a New Upgrade Handler](#1-prepare-a-new-upgrade-handler)
- [2. Create a GitHub Release](#2-create-a-github-release)
- [3. Prepare the Upgrade Transactions](#3-prepare-the-upgrade-transactions)
- [4. \[Complex Upgrades Only\] Test the New Release Locally](#4-complex-upgrades-only-test-the-new-release-locally)
- [5. Submit the Upgrade on each network](#5-submit-the-upgrade-on-each-network)
- [6. Update the release notes](#6-update-the-release-notes)
  - [6.1 Update the GitHub Release Notes](#61-update-the-github-release-notes)
  - [6.2 Update the Documentation Upgrade List](#62-update-the-documentation-upgrade-list)
- [7. Update the `homebrew-tap` Formula](#7-update-the-homebrew-tap-formula)
- [8. Communicate the Update](#8-communicate-the-update)
  - [8.1 Community Discord Server](#81-community-discord-server)
  - [8.2 CEXs on Telegram](#82-cexs-on-telegram)
- [9. Troubleshooting \& Canceling an Upgrade](#9-troubleshooting--canceling-an-upgrade)
- [9. Finish off checklist](#9-finish-off-checklist)
- [TODOs \& Improvements](#todos--improvements)

## 0. Communicate

Start a discord thread similar to [this v0.1.21 thread](https://discord.com/channels/824324475256438814/1384985059873918986) to communicate updates along the way in case of any issues.

## 1. Prepare a New Upgrade Handler

1. Identify the version of the [latest release](https://github.com/pokt-network/poktroll/releases/latest) from the [full list of releases](https://github.com/pokt-network/poktroll/releases) (e.g. `v0.1.20`)
2. Prepare a new upgrade handler by copying `vNEXT.go` to the next release (e.g. `v0.1.21`) like so:

   ```bash
   cp app/upgrades/vNEXT.go app/upgrades/v0.1.21.go
   ```

3. Open `v0.1.21.go` and replace all instances of `vNEXT` with `v0.1.21`.
4. Remove all the general purpose template comments from `v0.1.21.go`.
5. Open `app/upgrades.go` and:

   - Comment out the old upgrade
   - Add the new upgrade to `allUpgrades`

6. Prepare a new `vNEXT.go` by copying `vNEXT_Template.go` to `vNEXT.go` like so:

   ```bash
   cp app/upgrades/vNEXT_Template.go app/upgrades/vNEXT.go
   ```

7. Open `vNEXT.go` and remove all instances of `Template`.
8. Create a PR with these changes and merge it. ([Example](https://github.com/pokt-network/poktroll/pull/1520)):

```bash
 git commit -am "Adding v0.1.21.go upgrade handler"
 git push
```

## 2. Create a GitHub Release

1. **Tag the release** using one of the following and follow on-screen prompts:

   ```bash
   make release_tag_bug_fix
   # OR
   make release_tag_minor_release
   ```

2. **Publish the release** by:

   - Following the onscreen instructions from `make target` (e.g. pushing the tag)
   - [Drafting a new release](https://github.com/pokt-network/poktroll/releases/new)
   - Use the tag above to auto-generate the release notes

3. **Set as a pre-release** (change to `latest release` after upgrade completes).

:::note 😎 Keep Calm and Wait for CI 😅

Wait for the [`Release Artifacts`](https://github.com/pokt-network/poktroll/actions/workflows/release-artifacts.yml) CI job to build artifacts for your release.

It'll take ~20 minutes and will be auto-attached to the release under the `Assets` section once complete.

:::

## 3. Prepare the Upgrade Transactions

Generate the new upgrade transaction JSON files like so:

```bash
./tools/scripts/upgrades/prepare_upgrade_tx.sh v<YOUR_VERSION>.<YOUR_RELEASE>.<YOUR_PATCH>
```

For example:

```bash
./tools/scripts/upgrades/prepare_upgrade_tx.sh v0.1.21
```

This will create:

```bash
tools/scripts/upgrades/upgrade_tx_v0.1.21_alpha.json
tools/scripts/upgrades/upgrade_tx_v0.1.21_beta.json
tools/scripts/upgrades/upgrade_tx_v0.1.21_local.json
tools/scripts/upgrades/upgrade_tx_v0.1.21_main.json
```

_Note that the `height` is not populated in the `*.json` files. This will be updated in subsequent steps below._

<details>
<summary>Example JSON snippet:</summary>

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

</details>

<details>

<summary>**Optional**: Validate the Upgrade Binary URLs</summary>

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

</details>

## 4. [Complex Upgrades Only] Test the New Release Locally

:::warning Chain Halt Risk

If your upgrade handle had complex business logic, you MUST test it locally to avoid a chain halt.

:::

Follow [Testing Protocol Upgrades](3_testing_upgrades_locally.md) **BEFORE** submitting any transactions.

If you find an issue, you'll need to:

1. Delete the previous release
2. Delete the previous tag
3. Implement and merge in the fix
4. Prepare a new release
5. Regenerate the artifacts
6. Repeat the process above

## 5. Submit the Upgrade on each network

:::note Familiarize yourself with the playbook

If this is your first time submitting a tx upgrade, run the following command and
familiarize yourself with the playbook that gets generated before procedding:

```bash
./tools/scripts/upgrades/submit_upgrade.sh beta v0.42.69
```

Then proceed to generate the real commands.

:::

If you are submitting the upgrade for `v0.1.21`, follow the instructions
generated by the `prepare_upgrade_tx.sh` script for each environment.

```bash
./tools/scripts/upgrades/submit_upgrade.sh alpha v0.1.21
./tools/scripts/upgrades/submit_upgrade.sh beta v0.1.21
./tools/scripts/upgrades/submit_upgrade.sh main v0.1.21
```

**Make sure to ONLY move to the next network after the prior one finished successfully.**

## 6. Update the release notes

### 6.1 Update the GitHub Release Notes

1. Generate a table of the upgrade heights and tx hashes like so:

   ```bash
   ./tools/scripts/upgrades/prepare_upgrade_release_notes.sh v0.1.21
   ```

2. Insert the table above the auto-generated [release notes](https://github.com/pokt-network/poktroll/releases).
3. Mark it as `latest release`.

### 6.2 Update the Documentation Upgrade List

Update the [Upgrade List Documentation](./4_upgrade_list.md) with the new upgrade.

Use the release notes

## 7. Update the `homebrew-tap` Formula

Once the upgrade is validated, update the tap so users can install the new CLI.

**Steps:**

```bash
git clone git@github.com:pokt-network/homebrew-pocketd.git
cd homebrew-pocketd
make tap_update_version
git commit -am "Update pocket tap from v.0.1.20 to v.0.1.21"
git push
```

_Note: Make sure to update `v0.1.20` and `v0.1.21` in the commit message above._

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

## 8. Communicate the Update

### 8.1 Community Discord Server

You can use the following template as a starting point for your release announcement.

In particular, call out:

- Call to action to the community with the new release
- Major new features or changes
- Thank everyone for their support and whoever was involved

Publish the announcements:

- [Discord Beta TestNet Announcement Channel](https://discord.com/channels/553741558869131266/1384589692355477696)
- [Discord MainNet Announcement Channel](https://discord.com/channels/553741558869131266/1384589604153331732)

<details>
<summary>Example Release Template From v0.1.20</summary>

@MWG :mega: The latest release is live! :mega:

The `v0.1.20` release is officially live on Alpha, Beta and MainNet!

✍ You can find the full release notes at : [poktroll/releases/tag/v0.1.20](https://github.com/pokt-network/poktroll/releases/tag/v0.1.20).

💾 The latest snapshot has been generated and uploaded [here](https://snapshots.us-nj.poktroll.com).

📸 Call to action: Start claiming your suppliers on MainNet!

❗ If you encounter any issues or bugs, please open up a [github issue](https://github.com/pokt-network/poktroll/issues/new/choose) for things that are important. For anything urgent: tag us here @Grove :herb:.

🙏 Thank you for all the feedback and cooperation. We're all on the cusp of something really big here. :chart_with_upwards_trend:

---

🔜 Just a heads up that `v0.1.21` will come out within the next 1-3 days!

</details>

### 8.2 CEXs on Telegram

You can use the following template as a starting point for your release announcement.

In particular, call out:

- Any other upcoming releases in the near future
- Provide support if they need help running a node

<details>
<summary>Example Release Template From v0.1.20</summary>

📣 The latest release is live! 📣

The `v0.1.20` release is officially live on Alpha, Beta and MainNet!

✍ You can find the full release notes at : [poktroll/releases/tag/v0.1.20](https://github.com/pokt-network/poktroll/releases/tag/v0.1.20).

💾 The latest snapshot has been generated and uploaded [here](https://snapshots.us-nj.poktroll.com/).

❗ If you encounter any issues or bugs, please open up a [GitHub issue](https://github.com/pokt-network/poktroll/issues/new/choose) for important things or tag us here.

Thanks in advance! 🙏

---

🔜 Just a heads up that `v0.1.21` will come out within the next 1-3 days!

---

**Endpoint Reminder** - As a reminder, you can use our public endpoint if you choose to: [https://shannon-grove-rpc.mainnet.poktroll.com/](https://shannon-grove-rpc.mainnet.poktroll.com/), which is always up to date.

If you use the [Cosmos SDK cosmovisor](https://docs.cosmos.network/main/build/tooling/cosmovisor), or the [full node script built by Grove](https://dev.poktroll.com/operate/cheat_sheets/full_node_cheatsheet), your nodes should have automatically upgraded.

**🌿 Open Offer from Grove** - As a close partner, we’re also happy to spin up a dedicated endpoint for your exchange specifically. Just let us know if you ever need this in the future!

</details>

## 9. Troubleshooting & Canceling an Upgrade

- 🌿 Grove Only: [Infrastructure Helper Scripts](https://github.com/buildwithgrove/infrastructure/tree/main/scripts)
- [Chain Halt Troubleshooting](./7_chain_halt_troubleshooting.md)
- [Failed upgrade contingency plan](./8_contigency_plans.md)
- [Chain Halt Recovery](./9_recovery_from_chain_halt.md)

## 9. Finish off checklist

- [ ] Update the [Upgrade List](./4_upgrade_list.md)
- [ ] [Create snapshot](https://www.notion.so/buildwithgrove/Shannon-Snapshot-Playbook-1aea36edfff680bbb5a7e71c9846f63c?source=copy_link) for each network

## TODOs & Improvements

- [ ] Concrete examples of PR examples & descriptions along the way
- [ ] Remind the reader to make the release the latest at the very end.
- [ ] Add dashboard links to observability
