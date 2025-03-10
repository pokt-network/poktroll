---
title: Upgrade procedure
sidebar_position: 2
---

# Upgrade procedure <!-- omit in toc -->

:::warning

This page describes the protocol upgrade process, intended for the protocol team's internal use.

If you're interested in upgrading your Pocket Network node, please check our [releases page](https://github.com/pokt-network/poktroll/releases) for upgrade instructions and changelogs.

:::

- [When is an Upgrade Warranted?](#when-is-an-upgrade-warranted)
- [Implementing the Upgrade](#implementing-the-upgrade)
- [Writing an Upgrade Transaction](#writing-an-upgrade-transaction)
  - [Validate the URLs (live network only)](#validate-the-urls-live-network-only)
- [Submitting the upgrade onchain](#submitting-the-upgrade-onchain)
- [Cancelling the upgrade plan](#cancelling-the-upgrade-plan)
- [Testing the Upgrade](#testing-the-upgrade)
  - [LocalNet Upgrades](#localnet-upgrades)
    - [LocalNet Upgrade Cheat Sheet](#localnet-upgrade-cheat-sheet)
  - [DevNet Upgrades](#devnet-upgrades)
  - [TestNet Upgrades](#testnet-upgrades)
  - [Mainnet Upgrades](#mainnet-upgrades)

## Overview <!-- omit in toc -->

When a consensus-breaking change is made to the protocol, we must carefully evaluate and implement an upgrade path that
allows existing nodes to transition safely from one software version to another without disruption.

This process involves several key steps:

1. **Proposal**: The DAO drafts an upgrade proposal using our offchain governance system.
2. **Implementation**: The proposed changes are implemented in the codebase.
3. **Testing**: Thorough testing of the proposed changes is conducted in devnet and testnet environments before mainnet deployment.
4. **Announcement**: Upon successful testing, we announce the upgrade through our social media channels and community forums.
5. **Deployment**: An upgrade transaction is sent to the network, allowing node operators using [Cosmovisor](../../operate/walkthroughs/full_node_walkthrough.md) to automatically upgrade their nodes at the specified block height.
6. **Monitoring**: Post-deployment, we closely monitor the network to ensure everything functions as expected.

## When is an Upgrade Warranted?

An upgrade is necessary whenever there's an API, State Machine, or other Consensus breaking change in the version we're about to release.

## Implementing the Upgrade

1. When a new version includes a `consensus-breaking` change, plan for the next protocol upgrade:

   - If there's a change to a specific module -> bump that module's consensus version.
   - Note any potential parameter changes to include in the upgrade.

2. Create a new upgrade in `app/upgrades`:
   - Refer to `historical.go` for past upgrades and examples.
   - Consult Cosmos-sdk documentation on upgrades for additional guidance on [building-apps/app-upgrade](https://docs.cosmos.network/main/build/building-apps/app-upgrade) and [modules/upgrade](https://docs.cosmos.network/main/build/modules/upgrade).

3. Update the `app/upgrades.go` file to include the new upgrade plan in `allUpgrades`.

:::info

Creating a new upgrade plan **MUST BE DONE** even if there are no state changes.

:::

## Writing an Upgrade Transaction

An upgrade transaction includes a [Plan](https://github.com/cosmos/cosmos-sdk/blob/0fda53f265de4bcf4be1a13ea9fad450fc2e66d4/x/upgrade/proto/cosmos/upgrade/v1beta1/upgrade.proto#L14) with specific details about the upgrade.

This information helps schedule the upgrade on the network and provides necessary data for automatic upgrades via `Cosmovisor`.

A typical upgrade transaction includes:

- `name`: Name of the upgrade. It should match the `VersionName` of `upgrades.Upgrade`.
- `height`: The height at which an upgrade should be executed and the node will be restarted.
- `info`: Can be empty. **Only needed for live networks where we want cosmovisor to upgrade nodes automatically**.

And looks like the following as an example:

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
          "info": "{\"binaries\":{\"linux/amd64\":\"https://github.com/pokt-network/poktroll/releases/download/v0.0.4/poktroll_linux_amd64.tar.gz?checksum=sha256:49d2bcea02702f3dcb082054dc4e7fdd93c89fcd6ff04f2bf50227dacc455638\",\"linux/arm64\":\"https://github.com/pokt-network/poktroll/releases/download/v0.0.4/poktroll_linux_arm64.tar.gz?checksum=sha256:698f3fa8fa577795e330763f1dbb89a8081b552724aa154f5029d16a34baa7d8\",\"darwin/amd64\":\"https://github.com/pokt-network/poktroll/releases/download/v0.0.4/poktroll_darwin_amd64.tar.gz?checksum=sha256:5ecb351fb2f1fc06013e328e5c0f245ac5e815c0b82fb6ceed61bc71b18bf8e9\",\"darwin/arm64\":\"https://github.com/pokt-network/poktroll/releases/download/v0.0.4/poktroll_darwin_arm64.tar.gz?checksum=sha256:a935ab83cd770880b62d6aded3fc8dd37a30bfd15b30022e473e8387304e1c70\"}}"
        }
      }
    ]
  }
}
```

:::tip

When `cosmovisor` is configured to automatically download binaries, it will pull the binary from the link provided in
the object about and perform a hash verification (which is also optional).

**NOTE THAT** we only know the hashes **AFTER** the release has been cut and CI created artifacts for this version.

:::

### Validate the URLs (live network only)

The URLs of the binaries contain checksums. It is critical to ensure they are correct.
Otherwise Cosmovisor won't be able to download the binaries and go through the upgrade.

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

## Submitting the upgrade onchain

The `MsgSoftwareUpgrade` can be submitted using the following command:

```bash
poktrolld tx authz exec $PATH_TO_UPGRADE_TRANSACTION_JSON --from=pnf
```

If the transaction has been accepted, the upgrade plan can be viewed with this command:

```bash
poktrolld query upgrade plan
```

## Cancelling the upgrade plan

It is possible to cancel the upgrade before the upgrade plan height is reached. To do so, execute the following make target:

```bash
make localnet_cancel_upgrade
```

## Testing the Upgrade

:::warning
Note that for local testing, `cosmovisor` won't pull the binary from the upgrade Plan's info field.
:::

### LocalNet Upgrades

LocalNet **DOES NOT** support `cosmovisor` and automatic upgrades at the moment.

However, **IT IS NOT NEEDED** to simulate and test the upgrade procedure.

#### LocalNet Upgrade Cheat Sheet

For a hypothetical scenario to upgrade from `0.1` to `0.2`:

1. **Stop LocalNet** to prevent interference. Pull the `poktroll` repo into two separate directories. Let's name them `old` and `new`. It is recommended to open at least two tabs/shell panels in each directory for easier switching between directories.

2. **(`old` repo)** - Check out the old version. For the test to be accurate, we need to upgrade from the correct version.

   ```bash
   git checkout v0.1
   ```

3. **(`new` repo)**

   ```bash
   git checkout -b branch_to_test
   ```

   Replace `branch_to_test` with the actual branch you want to test.

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
   rm -rf ~/.poktroll && ./release_binaries/poktroll_darwin_arm64 comet unsafe-reset-all && make localnet_regenesis
   ```

6. **(`old` repo)** Write and save [an upgrade transaction](#writing-an-upgrade-transaction) for `v0.2`. The upgrade plan should be named after the version to which you're upgrading.

7. **(`old` repo)** Start the node:

   ```bash
   ./release_binaries/poktroll_darwin_arm64 start
   ```

   The validator node should run and produce blocks as expected.

8. **(`old` repo)** Submit the upgrade transaction. **NOTE THAT** the upgrade height in the transaction should be higher than the current block height. Adjust and submit if necessary:

   ```bash
   ./release_binaries/poktroll_darwin_arm64 tx authz exec tools/scripts/upgrades/local_test_v0.2.json --from=pnf
   ```

   Replace the path to the JSON transaction with your prepared upgrade transaction. Verify the upgrade plan was submitted and accepted:

   ```bash
   ./release_binaries/poktroll_darwin_arm64 query upgrade plan
   ```

9. Wait for the upgrade height to be reached on the old version. The old version should stop working since it has no knowledge of the `v0.2` upgrade. This simulates a real-world scenario. Stop the old node, and switch to the new version.

10. **(`new` repo)**

    ```bash
    ./release_binaries/poktroll_darwin_arm64 start
    ```

11. **Observe the output:**

    - A successful upgrade should output `applying upgrade "v0.2" at height: 20 module=x/upgrade`.
    - The node on the new version should continue producing blocks.
    - If there were errors during the upgrade, investigate and address them.

12. **(`new` repo, optional**) - If parameters were changed during the upgrade, test if these changes were applied. For example:

    ```bash
    ./release_binaries/poktroll_darwin_arm64 q application params
    ```

### DevNet Upgrades

DevNets currently do not support `cosmovisor`.

We use Kubernetes to manage software versions, including validators. Introducing another component to manage versions would be complex, requiring a re-architecture of our current solution to accommodate this change.

### TestNet Upgrades

We currently deploy TestNet validators using Kubernetes with helm charts, which prevents us from managing the validator with `cosmovisor`. We do not control what other TestNet participants are running. However, if participants have deployed their nodes using the [cosmovisor guide](../../operate/walkthroughs/full_node_walkthrough.md), their nodes will upgrade automatically.

Until we transition to [cosmos-operator](https://github.com/strangelove-ventures/cosmos-operator), which supports scheduled upgrades (although not fully automatic like `cosmovisor`), we need to manually manage the process:

1. Estimate when the upgrade height will be reached.
2. When validator node(s) stop due to an upgrade, manually perform an ArgoCD apply and clean up old resources.
3. Monitor validator node(s) as they start and begin producing blocks.

:::tip

If you are a member of Grove, you can find the instructions to access the infrastructure [on notion](https://www.notion.so/buildwithgrove/How-to-re-genesis-a-Shannon-TestNet-a6230dd8869149c3a4c21613e3cfad15?pvs=4).

:::

### Mainnet Upgrades

The Mainnet upgrade process is to be determined. We aim to develop and implement improved tooling for this environment.
