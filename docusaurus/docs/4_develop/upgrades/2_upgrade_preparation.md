---
title: Upgrade Preparation
sidebar_position: 2
---

:::note Purpose of this document

Technical instructions on preparing a protocol upgrade.

Instructions to create a new upgrade and apply it to both Alpha & Beta TesNet.

**You MUST follow this PRIOR TO the [MainNet Release Procedure](./3_mainnet_release_procedure.md).**

:::

## Required Prerequisites Setup, Reading & Knowledge <!-- omit in toc -->

- Ensure you've read and understood [When is a Protocol Upgrade Needed?](./1_upgrade_overview.md) at least once
- Ensure you have push access to [pokt-network/poktroll](https://github.com/pokt-network/poktroll)
- Familiarize yourself with the [list of previous upgrade handlers](https://github.com/pokt-network/poktroll/tree/main/app/upgrades)
- Familiarize yourself with [how to test changes locally](./4_localnet_upgrade_testing.md)
- Ensure you have the required CLI tools: `git`, `make`, `jq`, `sed`, `curl`, `go`, `brew`, `pocketd`, etc.

## Table of Contents <!-- omit in toc -->

- [1. Avoid On-Chain Non-Determinism](#1-avoid-on-chain-non-determinism)
- [2. Prepare a New Upgrade Handler](#2-prepare-a-new-upgrade-handler)
- [3. Test Locally](#3-test-locally)
- [4. Create a GitHub Release](#4-create-a-github-release)
- [5. Prepare the Upgrade Transactions](#5-prepare-the-upgrade-transactions)
- [6. Prepare Snapshots](#6-prepare-snapshots)
- [7. Submit the Upgrade on Alpha \& Beta TestNet](#7-submit-the-upgrade-on-alpha--beta-testnet)
- [8. Troubleshooting \& Canceling an Upgrade](#8-troubleshooting--canceling-an-upgrade)

## 1. Avoid On-Chain Non-Determinism

**⚠️ The most common cause of chain halts is non-deterministic onchain behavior ⚠️**

No amount of code reviews or testing can fully catch this. Here is a suggested and opinionated potential solution:

1. Start a session of [Claude code](https://www.anthropic.com/claude-code) or CLI agent of choice
2. Identify the tag of the previous MainNet release (e.g. `v0.1.20`)
3. Ask Claude the following:

   ```bash
   You are a senior CosmosSDK protocol engineer in charge of the next protocol upgrade.

   Do a git diff v0.1.20.

   Identify any potential bugs, edge cases or issues.

   In particular, focus on any onchain behaviour that can result in non-deterministic outcomes. For example, iterating a map without sorting the keys first.

   This is critical to avoid chain halts. Take your time and provide a comprehensive analysis.
   ```

## 2. Prepare a New Upgrade Handler

1. Identify the version of the [latest release](https://github.com/pokt-network/poktroll/releases/latest) from the [full list of releases](https://github.com/pokt-network/poktroll/releases) (e.g. `v0.1.20`)
2. Prepare a new upgrade handler by copying `vNEXT.go` to the next release (e.g. `v0.1.21`) like so:

   ```bash
   cp app/upgrades/vNEXT.go app/upgrades/v0.1.21.go
   ```

3. Open `app/upgrades/v0.1.21.go` and:
   - Replace all instances of `vNEXT` with `v0.1.21`.
   - Replace all instances of `_NEXT_` with `_0_1_21_`.
4. Remove all the general purpose template comments from `v0.1.21.go`.
5. Open `app/upgrades.go` and:

   - Comment out the old upgrade
   - Add the new upgrade to `allUpgrades`

6. Prepare a new `vNEXT.go` by copying `vNEXT_Template.go` to `vNEXT.go` like so:

   ```bash
   cp app/upgrades/vNEXT_Template.go app/upgrades/vNEXT.go
   ```

7. Open `app/upgrades/vNEXT.go` and remove all instances of `Template`.
8. Create a PR with these changes and merge it. [v0.1.21 PR Example](https://github.com/pokt-network/poktroll/pull/1520):
9. Note that the upgrade handler MAY need business logic related to onchain state changes. See [other upgrades](https://github.com/pokt-network/poktroll/tree/main/app/upgrades) for reference.

## 3. Test Locally

Follow the instructions in [Testing Protocol Upgrades Locally](4_localnet_upgrade_testing.md).

<details>
<summary>Expertise required for complex upgrades</summary>

⚠️ If your upgrade handle had complex business logic, you MUST test it locally to avoid a chain halt. ⚠️

Follow [Testing Protocol Upgrades](4_localnet_upgrade_testing.md) **BEFORE** submitting any transactions.

If you find an issue, you'll need to:

1. Delete the previous release
2. Delete the previous tag
3. Implement and merge in the fix
4. Prepare a new release
5. Regenerate the artifacts
6. Repeat the process above

This requires jumping back and forth between some of the steps on this page.

</details>

## 4. Create a GitHub Release

1. **Tag the release** by running the following command:

   ```bash
   make release_tag_minor
   ```

2. **Publish the release** by:

   - Following the onscreen instructions after running the command above
   - [Drafting a new release](https://github.com/pokt-network/poktroll/releases/new)
   - Use the tag above to auto-generate the release notes

3. **Set as a pre-release** (change to `latest release` after upgrade completes).
4. Trigger the workflow to build new release artifacts by running:

   ```bash
   gh workflow run "Release artifacts"
   ```

   :::note 😎 Keep Calm and Wait for CI 😅

   Wait for the [`Release Artifacts`](https://github.com/pokt-network/poktroll/actions/workflows/release-artifacts.yml) CI job to build artifacts for your release.

   It'll take ~20 minutes and will be auto-attached to the release under the `Assets` section once complete.

   :::

## 5. Prepare the Upgrade Transactions

Generate the new upgrade transaction JSON files. For example, for `v0.1.21`, run:

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

**Make sure to commit these to GitHub once you're done.**

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

## 6. Prepare Snapshots

Generate new snapshots for each network and ensure they are available [here](https://snapshots.us-nj.poktroll.com/).

:::warning Manual Process by Grove 🌿

This is currently a manual process maintained by the team at [Grove](https://grove.city).

The instructions are currently maintained in an [internal Notion](https://www.notion.so/buildwithgrove/Shannon-Snapshot-Playbook-1aea36edfff680bbb5a7e71c9846f63c?source=copy_link) document.

:::

## 7. Submit the Upgrade on Alpha & Beta TestNet

If you are submitting the upgrade for `v0.1.21`, follow the instructions generated by these commands:

```bash
./tools/scripts/upgrades/submit_upgrade.sh alpha v0.1.21
./tools/scripts/upgrades/submit_upgrade.sh beta v0.1.21
```

**⚠️ Make sure to ONLY move to the next network after the prior one finished successfully ⚠️**

## 8. Troubleshooting & Canceling an Upgrade

- [Infrastructure Documentation](https://github.com/buildwithgrove/infrastructure/tree/main/docs); 🌿 Grove Only
- [Chain Halt Troubleshooting](./7_chain_halt_troubleshooting.md)
- [Failed upgrade contingency plan](8_chain_halt_upgrade_contigency_plans.md)
- [Chain Halt Recovery](9_chain_halt_recovery.md)
