---
title: Protocol Upgrade Procedure
sidebar_position: 3
---

:::warning
This document is intended for core protocol developers.

We recommend reviewing the `Testing An Upgrade` section below to ensure that the upgrade process is completed successfully.
:::

## When is an Protocol Upgrade Warranted? <!-- omit in toc -->

A protocol upgrade is **required** if there are `consensus-breaking` changes. _For example, changes to protobufs._

A protocol upgrade is **optional but recommended** if there are changes to node software but are not `consensus-breaking` changes. _For examples, performance improvements._

A software release **can be made** with or without a protocol upgrade. _For example, new utilities in the CLI._

**Identify consensus breaking changes** by:

1. Reviewing merged [Pull Requests (PRs) with the `consensus-breaking` label](https://github.com/pokt-network/poktroll/issues?q=label%3Aconsensus-breaking+) since the last release. It is not a source of truth, but directionality correct.
2. Looking for breaking changes in `.proto` files
3. Looking for breaking changes in the `x/` directories
4. Identifying new onchain parameters or authorizations

:::info Non-exhaustive list

Note that the above is a non-exhaustive list and requires protocol expertise to identify all potential `consensus-breaking` changes.
:::

## Table of Contents <!-- omit in toc -->

- [Process Overview](#process-overview)
- [A. Implementing the Upgrade](#a-implementing-the-upgrade)
  - [1. Bump Module Consensus Version](#1-bump-module-consensus-version)
  - [2. Prepare a New Upgrade](#2-prepare-a-new-upgrade)
  - [3. Write an Upgrade Transaction](#3-write-an-upgrade-transaction)
  - [4. Validate the Binary URLs (live network only)](#4-validate-the-binary-urls-live-network-only)
  - [5. Submit the Upgrade Onchain](#5-submit-the-upgrade-onchain)
  - [6. \[Optional\] Cancel the Upgrade Plan (if needed)](#6-optional-cancel-the-upgrade-plan-if-needed)
- [B. Testing the Upgrade](#b-testing-the-upgrade)
  - [LocalNet Upgrades](#localnet-upgrades)
  - [DevNet Upgrades](#devnet-upgrades)
  - [TestNet Upgrades](#testnet-upgrades)
    - [TestNet Management - Grove Employees](#testnet-management---grove-employees)
    - [Alpha TestNet](#alpha-testnet)

## Process Overview

When a `consensus-breaking` change is made to the protocol, we must carefully evaluate and implement an upgrade path that
allows existing nodes to transition safely from one software version to another without disruption.

This process involves several key steps:

1. **Proposal**: The DAO drafts an upgrade proposal using our offchain governance system.
2. **Implementation**: The proposed changes are implemented in the codebase.
3. **Testing**: Thorough testing of the proposed changes is conducted in devnet and testnet environments before mainnet deployment.
4. **Announcement**: Upon successful testing, we announce the upgrade through our social media channels and community forums.
5. **Deployment**: An upgrade transaction is sent to the network, allowing node operators using [Cosmovisor](../walkthroughs/full_node_walkthrough.md) to automatically upgrade their nodes at the specified block height.
6. **Monitoring**: Post-deployment, we closely monitor the network to ensure everything functions as expected.

## A. Implementing the Upgrade

### 1. Bump Module Consensus Version

If there's a change to a specific module, bump that module's [ConsensusVersion](https://github.com/search?q=repo%3Apokt-network%2Fpoktroll%20ConsensusVersion&type=code).

### 2. Prepare a New Upgrade

:::warning MUST BE DONE

Creating a new upgrade plan **MUST BE DONE** even if there are no state changes.

:::

1. Review all [previous upgrades](https://github.com/pokt-network/poktroll/tree/main/app/upgrades) for reference.
   - Refer to `historical.go` for past upgrades and examples.
   - Consult Cosmos-sdk documentation on upgrades for additional guidance on [building-apps/app-upgrade](https://docs.cosmos.network/main/build/building-apps/app-upgrade) and [modules/upgrade](https://docs.cosmos.network/main/build/
2. Note any parameter changes, authorizations, functions or other state changes.
3. If modifying protobuf definitions, consider using the approach in [protobuf deprecation](5_protobuf_upgrades.md) for backward compatibility.
4. Update the `app/upgrades.go` file to include the new upgrade plan in `allUpgrades`.

### 3. Write an Upgrade Transaction

An upgrade transaction includes a [Plan](https://github.com/cosmos/cosmos-sdk/blob/0fda53f265de4bcf4be1a13ea9fad450fc2e66d4/x/upgrade/proto/cosmos/upgrade/v1beta1/upgrade.proto#L14) with specific details about the upgrade.

This information helps schedule the upgrade on the network and provides necessary data for automatic upgrades via `Cosmovisor`.

A typical upgrade transaction includes:

- `name`: Name of the upgrade. It should match the `VersionName` of `upgrades.Upgrade`.
- `height`: The height at which an upgrade should be executed and the node will be restarted.
- `info`: Can be empty. **Only needed for live networks where we want cosmovisor to upgrade nodes automatically**.

Here is an example for reference:

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
          "info": "{\"binaries\":{\"linux/amd64\":\"https://github.com/pokt-network/poktroll/releases/download/v0.0.4/poktroll_linux_amd64.tar.gz?checksum=sha256:49d2bcea02702f3dcb082054dc4e7fdd93c89fcd6ff04f2bf50227dacc455638\",\"linux/arm64\":\"https://github.com/pokt-network/poktroll/releases/download/v0.0.4/poktroll_linux_arm64.tar.gz?checksum=sha256:698f3fa8fa577795e330763f1dbb89a8081b552724aa154f5029d16a34baa7d8\",\"darwin/amd64\":\"https://github.com/pokt-network/poktroll/releases/download/v0.0.4/pocket_darwin_amd64.tar.gz?checksum=sha256:5ecb351fb2f1fc06013e328e5c0f245ac5e815c0b82fb6ceed61bc71b18bf8e9\",\"darwin/arm64\":\"https://github.com/pokt-network/poktroll/releases/download/v0.0.4/pocket_darwin_arm64.tar.gz?checksum=sha256:a935ab83cd770880b62d6aded3fc8dd37a30bfd15b30022e473e8387304e1c70\"}}"
        }
      }
    ]
  }
}
```

:::tip

When `cosmovisor` is configured to automatically download binaries, it will pull the binary from the link provided in
the upgrade object and perform a hash verification (optional).

**NOTE THAT** we only know the hashes **AFTER** the release has been cut and CI created artifacts for this version.

:::

### 4. Validate the Binary URLs (live network only)

The URLs of the binaries contain checksums. It is critical to ensure they are correct.
**Otherwise, Cosmovisor won't be able to download the binaries and go through the upgrade.**

The command below (using tools build by the authors of Cosmosvisor) can be used to achieve the above:

```bash
jq -r '.body.messages[0].plan.info | fromjson | .binaries[]' $PATH_TO_UPGRADE_TRANSACTION_JSON | while IFS= read -r url; do
  go-getter "$url" .
done
```

The output should look like this:

```text
2024/09/24 12:40:40 success!
2024/09/24 12:40:42 success!
2024/09/24 12:40:44 success!
2024/09/24 12:40:46 success!
```

:::tip

`go-getter` can be installed using the following command:

```bash
go install github.com/hashicorp/go-getter/cmd/go-getter@latest
```

:::

### 5. Submit the Upgrade Onchain

The `MsgSoftwareUpgrade` can be submitted using the following command:

```bash
pocketd tx authz exec $PATH_TO_UPGRADE_TRANSACTION_JSON --from=pnf
```

If the transaction has been accepted, the upgrade plan can be viewed with this command:

```bash
pocketd query upgrade plan
```

### 6. [Optional] Cancel the Upgrade Plan (if needed)

It is possible to cancel the upgrade before the upgrade plan height is reached.

To do so, execute the following make target:

```bash
make localnet_cancel_upgrade
```

## B. Testing the Upgrade

### LocalNet Upgrades

:::warning

LocalNet **DOES NOT** support `cosmovisor` and automatic upgrades at the moment. `cosmosvisor` doesn't pull the binary from the upgrade Plan's info field.

However, **IT IS NOT NEEDED** to simulate and test the upgrade procedure.

:::

Below is a set of instructions for a hypothetical upgrade from `0.1` to `0.2`:

1. **Stop LocalNet** to prevent interference. Use `git worktree` to check out the `poktroll` repo to a different branch/tag in a separate directory. We'll refer to the old and new branches as `old` and `new` respectively. It is recommended to open at least two tabs/shell panels in each directory for easier switching between directories.

2. **(`old` branch)** - Use `git worktree` to check out the old version in a new directory. For the test to be accurate, we need to upgrade from the correct version.

   ```bash
   git worktree add ../poktroll-old v0.1
   ```

   :::tip Cleaning Up

   When you're finished and ready to remove the `old` worktree (the new directory associated with the old branch):

   ```bash
   git worktree remove ../poktroll-old
   ```

   This won't have any effect on the git repo itself, nor on the default worktree (unstaged/uncommitted changes, stash, etc.).

   :::

3. **(`new` branch)**

   ```bash
   git checkout -b branch_to_test
   ```

   Replace `branch_to_test` with the actual branch or tag that you want to test.

   :::note
   This branch should have an upgrade implemented per the docs in [Implementing the Upgrade](#implementing-the-upgrade).
   Here, the upgrade should be named `v0.2`.
   :::

4. **(BOTH repos)** - We'll use binaries from both versions - old and new.

   ```bash
   make go_develop ignite_release ignite_release_extract_binaries
   ```

   :::note
   The binary produced by these commands in the old repo should result in the same binary as it was downloaded from [production releases](https://github.com/pokt-network/poktroll/releases). You can use them as an alternative to building the binary from source.
   :::

5. **(`old` repo)** - Clean up and generate an empty genesis using the old version.

   ```bash
   rm -rf ~/.pocket && ./release_binaries/pocket_darwin_arm64 comet unsafe-reset-all && make localnet_regenesis
   ```

6. **(`old` repo)** Start the node:

   ```bash
   ./release_binaries/pocket_darwin_arm64 start
   ```

   The validator node should run and produce blocks as expected.

7. **(`old` repo)** Submit the upgrade transaction. **NOTE THAT** the upgrade height in the transaction should be higher than the current block height. Adjust and submit if necessary:

   ```bash
   ./release_binaries/pocket_darwin_arm64 tx authz exec tools/scripts/upgrades/local_test_v0.2.json --from=pnf
   ```

   Replace the path to the JSON transaction with your prepared upgrade transaction. Verify the upgrade plan was submitted and accepted:

   ```bash
   ./release_binaries/pocket_darwin_arm64 query upgrade plan
   ```

8. **(`old` repo)** - Wait for the upgrade height to be reached on the old version. The old version should stop working since it has no knowledge of the `v0.2` upgrade. This simulates a real-world scenario. Stop the old node, and switch to the new version.

9. **(`new` repo)**

   ```bash
   ./release_binaries/pocket_darwin_arm64 start
   ```

10. **(`new` repo)** - Observe the output:

    - A successful upgrade should output `applying upgrade "v0.2" at height: 20 module=x/upgrade`.
    - The node on the new version should continue producing blocks.
    - If there were errors during the upgrade, investigate and address them.

11. **(`new` repo, optional**) - If parameters were changed during the upgrade, test if these changes were applied. For example:

    ```bash
    ./release_binaries/pocket_darwin_arm64 q application params
    ```

### DevNet Upgrades

**DevNets do not currently support `cosmovisor`.**

We use Kubernetes to manage software versions, including validators. Introducing another component to manage versions would be complex, requiring a re-architecture of our current solution to accommodate this change.

### TestNet Upgrades

Participants have deployed their full nodes using the [cosmovisor guide](../walkthroughs/full_node_walkthrough.md) will have upgrade automatically.

Participants who do not use `cosmosvisor` will need to manually manage the process by:

1. Estimating when the upgrade height will be reached
2. When validator node(s) stop due to an upgrade, manually perform an update (e.g. ArgoCD apply and clean up old resources)
3. Monitor full & validator node(s) as they start and begin producing blocks.

:::note TODO: Cosmos Operator

[cosmos-operator](https://github.com/strangelove-ventures/cosmos-operator) supports scheduled upgrades and is also an option if not using `cosmovisor`

:::

#### TestNet Management - Grove Employees

:::warning

This section is intended for Grove employees only who help manage & maintain TestNet Infrastructure.

:::

#### Alpha TestNet

There are two validators in linode. Three on vultr. One seed on vultr. No TestNet infra on GCP.

I think the only gotcha is as upgrade happens, cosmovisor backs up data dir on all nodes. So it might take a few minutes to finish that process before starting the node after upgrade.

There’s only dashboard for beta testnet. No one place to see the health of alpha.. logs are shipped to victoria logs but I always used k8s client instead.
