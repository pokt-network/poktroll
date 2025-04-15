---
title: Testing Protocol Upgrades
sidebar_position: 3
---

:::warning
This document is intended for core protocol developers.

We recommend reviewing the `Testing An Upgrade` section below to ensure that the upgrade process is completed successfully.
:::

## Table of Contents <!-- omit in toc -->

- [TestNet Upgrades](#testnet-upgrades)
  - [TestNet Management - Grove Employees](#testnet-management---grove-employees)
  - [Alpha TestNet](#alpha-testnet)

### Testing the Upgrade (Before Merging)

```
**Shell #1: Old software (that will listen on the upgrade)**

```bash
git clone git@github.com:pokt-network/poktroll.git poktroll_t2
cd poktroll_t2
gco v0.1.1 # Checkout tag of last release
make go_develop ignite_release ignite_release_extract_binaries
./release_binaries/pocket_darwin_arm64 start
```

```bash
make localnet_cancel_upgrade
```

We are using the `upgrade/migration` branch as an example, but make sure to update
it in the example below with your own branch

**Shell #1: New software (from where the upgrade will be issued)**

```bash
git clone git@github.com:pokt-network/poktroll.git poktroll_t1
cd poktroll_t1
git checkout -b upgrade/upgrade_v_0_1_2 origin/upgrade_v_0_1_2 # Checkout branch of new release
make go_develop ignite_release ignite_release_extract_binaries
./release_binaries/pocket_darwin_arm64 comet unsafe-reset-all && make localnet_regenesis
./release_binaries/pocket_darwin_arm64 start
./release_binaries/pocket_darwin_arm64 tx authz exec tools/scripts/upgrades/local_test_v1.0.2.json --from=pnf

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
   This branch should have an upgrade implemented per the docs in [Implementing the Upgrade].
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

## TestNet Upgrades

Participants have deployed their full nodes using the [cosmovisor guide](../walkthroughs/full_node_walkthrough.md) will have upgrade automatically.

Participants who do not use `cosmosvisor` will need to manually manage the process by:

1. Estimating when the upgrade height will be reached
2. When validator node(s) stop due to an upgrade, manually perform an update (e.g. ArgoCD apply and clean up old resources)
3. Monitor full & validator node(s) as they start and begin producing blocks.

:::note TODO: Cosmos Operator

[cosmos-operator](https://github.com/strangelove-ventures/cosmos-operator) supports scheduled upgrades and is also an option if not using `cosmovisor`

:::

### TestNet Management - Grove Employees

:::warning

This section is intended for Grove employees only who help manage & maintain TestNet Infrastructure.

:::

### Alpha TestNet

There are two validators in linode. Three on vultr. One seed on vultr. No TestNet infra on GCP.

I think the only gotcha is as upgrade happens, cosmovisor backs up data dir on all nodes. So it might take a few minutes to finish that process before starting the node after upgrade.

There’s only dashboard for beta testnet. No one place to see the health of alpha.. logs are shipped to victoria logs but I always used k8s client instead.
