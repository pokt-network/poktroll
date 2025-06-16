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

- [1. Prepare a New Upgrade Handler](#1-prepare-a-new-upgrade-handler)
- [2. Create a GitHub Release](#2-create-a-github-release)
- [3. Prepare the Upgrade Transactions](#3-prepare-the-upgrade-transactions)
- [4. Test the New Release Locally](#4-test-the-new-release-locally)
- [5. Submit the Upgrade on each network](#5-submit-the-upgrade-on-each-network)
- [7. Update the release notes](#7-update-the-release-notes)
- [8. Update the `homebrew-tap` Formula](#8-update-the-homebrew-tap-formula)
- [9. Troubleshooting \& Canceling an Upgrade](#9-troubleshooting--canceling-an-upgrade)
- [10. Finish off checklist](#10-finish-off-checklist)
- [TODOs \& Improvements](#todos--improvements)

## 1. Prepare a New Upgrade Handler

1. Identify the version of the last [release](https://github.com/pokt-network/poktroll/releases) (e.g. `v0.1.20`)
2. Prepare a new upgrade handler by copying `vNEXT.go` to the next release (e.g. `v0.1.21`) like so:

   ```bash
   cp app/upgrades/vNEXT.go app/upgrades/v0.1.21.go
   ```

3. Open `v0.1.21.go` and replace all instances of `vNEXT` with `v0.1.21`.
4. Open `app/upgrades.go` and add the new upgrade to `allUpgrades`, commenting out the old upgrade.
5. Prepare a new `vNEXT.go` by copying `vNEXT_Template.go` to `vNEXT.go` like so:

   ```bash
   cp app/upgrades/vNEXT_Template.go app/upgrades/vNEXT.go
   ```

6. Open `vNEXT.go` and remove all instances of `Template`.
7. Create a PR with these changes ([example](https://github.com/pokt-network/poktroll/pull/1489)) and merge it.

## 2. Create a GitHub Release

1. **Tag the release** using one of the following and follow on-screen prompts:

   ```bash
   make release_tag_bug_fix
   # or
   make release_tag_minor_release
   ```

2. **Publish the release** by:

   - [Drafting a new release](https://github.com/pokt-network/poktroll/releases/new)
   - Use the tag above to auto-generate the release notes

3. **Set as a pre-release** (change to `latest release` after upgrade completes).

:::note 😎 Keep Calm and Wait for CI 😅

Wait for the [`Release Artifacts`](https://github.com/pokt-network/poktroll/actions/workflows/release-artifacts.yml) CI job to build artifacts for your release.

It'll take ~20 minutes and will be auto-attached to the release once complete.

:::

## 3. Prepare the Upgrade Transactions

Generate the new upgrade transaction JSON files like so:

```bash
./tools/scripts/upgrades/prepare_upgrade_tx.sh v<YOUR_VERSION>.<YOUR_RELEASE>.<YOUR_PATCH>
```

This will create:

```bash
tools/scripts/upgrades/upgrade_tx_vX.Y.Z_alpha.json
tools/scripts/upgrades/upgrade_tx_vX.Y.Z_beta.json
tools/scripts/upgrades/upgrade_tx_vX.Y.Z_local.json
tools/scripts/upgrades/upgrade_tx_vX.Y.Z_main.json
```

For example:

```bash
./tools/scripts/upgrades/prepare_upgrade_tx.sh v0.1.20
```

_Note that the `height` is not populated in the `*.json` files. You will need to update the `height` before submitting each one. More on this later..._

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

## 4. Test the New Release Locally

:::warning Chain Halt Risk

If your upgrade handle had complex business logic, you MUST test it locally to avoid a chain halt.

:::

Follow [Testing Protocol Upgrades](3_testing_upgrades_locally.md) **BEFORE** submitting any transactions.

If you find an issue, you'll need to:

1. Delete the previous release
2. Delete the previous tag
3. Prepare a new release
4. Regenerate the artifacts

## 5. Submit the Upgrade on each network

If you are submitting the upgrade for `v0.1.21`, follow the instructions
generated by the `prepare_upgrade_tx.sh` script.

```bash
./tools/scripts/upgrades/submit_upgrade.sh alpha v0.1.21
./tools/scripts/upgrades/submit_upgrade.sh beta v0.1.21
./tools/scripts/upgrades/submit_upgrade.sh main v0.1.21
```

**Make sure to ONLY move to the next network after the prior one finished successfully.**

## 7. Update the release notes

Generate a table of the upgrade heights and tx hashes like so:

```bash
./tools/scripts/upgrades/prepare_upgrade_release_notes.sh v0.1.21
```

And insert the table above the auto-generated release notes.

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

## 9. Troubleshooting & Canceling an Upgrade

- 🌿 Grove Only: [Infrastructure Helper Scripts](https://github.com/buildwithgrove/infrastructure/tree/main/scripts)
- [Chain Halt Troubleshooting](./7_chain_halt_troubleshooting.md)
- [Failed upgrade contingency plan](./8_contigency_plans.md)
- [Chain Halt Recovery](./9_recovery_from_chain_halt.md)

## 10. Finish off checklist

- [ ] Update the [Upgrade List](./4_upgrade_list.md)
- [ ] [Create snapshot](https://www.notion.so/buildwithgrove/Shannon-Snapshot-Playbook-1aea36edfff680bbb5a7e71c9846f63c?source=copy_link) for each network

## TODOs & Improvements

- [ ] Concrete examples of PR examples & descriptions along the way
- [ ] Remind the reader to make the release the latest at the very end.
- [ ] Add dashboard links to observability
