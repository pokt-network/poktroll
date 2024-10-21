---
title: Upgrade procedure
sidebar_position: 2
---

# Upgrade procedure <!-- omit in toc -->

:::warning
This page describes the protocol upgrade process, which is internal to the protocol team. If you're interested in upgrading your Pocket Network node, please check our [releases page](https://github.com/pokt-network/poktroll/releases) for upgrade instructions and changelogs.
:::

- [When is an Upgrade Warranted?](#when-is-an-upgrade-warranted)
- [Implementing the Upgrade](#implementing-the-upgrade)
- [Writing an Upgrade Transaction](#writing-an-upgrade-transaction)
  - [Validate the URLs (live network only)](#validate-the-urls-live-network-only)
- [Submitting the upgrade on-chain](#submitting-the-upgrade-on-chain)
- [Cancelling the upgrade plan](#cancelling-the-upgrade-plan)
- [Testing the Upgrade](#testing-the-upgrade)
  - [LocalNet](#localnet)
    - [LocalNet Upgrade tl;dr](#localnet-upgrade-tldr)
    - [LocalNet Upgrade Full Example Walkthrough](#localnet-upgrade-full-example-walkthrough)
  - [DevNet](#devnet)
  - [TestNet](#testnet)
  - [Mainnet](#mainnet)

## Overview <!-- omit in toc -->

When a consensus-breaking change is made to the protocol, we must carefully evaluate and implement an upgrade path that allows existing nodes to transition safely from one software version to another without disruption. This process involves several key steps:

1. **Proposal**: The DAO drafts an upgrade proposal using our off-chain governance system.
2. **Implementation**: The proposed changes are implemented in the codebase.
3. **Testing**: Thorough testing of the proposed changes is conducted in devnet and testnet environments before mainnet deployment.
4. **Announcement**: Upon successful testing, we announce the upgrade through our social media channels and community forums.
5. **Deployment**: An upgrade transaction is sent to the network, allowing node operators using [Cosmovisor](../../operate/run_a_node/full_node_cosmovisor.md) to automatically upgrade their nodes at the specified block height.
6. **Monitoring**: Post-deployment, we closely monitor the network to ensure everything functions as expected.

## When is an Upgrade Warranted?

An upgrade is necessary whenever there's an API, State Machine, or other Consensus breaking change in the version we're about to release.

## Implementing the Upgrade

1. When a new version includes a consensus-breaking change, plan for the next protocol upgrade:
   - If there's a change to a specific module, bump that module's consensus version.
   - Note any potential parameter changes to include in the upgrade.
2. Create a new upgrade in `app/upgrades`. **THIS MUST BE DONE** even if there are no state changes.
   - Refer to `historical.go` for past upgrades and examples.
   - Consult Cosmos-sdk documentation on upgrades for additional guidance [here](https://docs.cosmos.network/main/build/building-apps/app-upgrade) and [here](https://docs.cosmos.network/main/build/modules/upgrade).

## Writing an Upgrade Transaction

An upgrade transaction includes a [Plan](https://github.com/cosmos/cosmos-sdk/blob/0fda53f265de4bcf4be1a13ea9fad450fc2e66d4/x/upgrade/proto/cosmos/upgrade/v1beta1/upgrade.proto#L14) with specific details about the upgrade. This information helps schedule the upgrade on the network and provides necessary data for automatic upgrades via `Cosmovisor`. A typical upgrade transaction will look like the following:

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

- `name`: Name of the upgrade. It should match the `VersionName` of `upgrades.Upgrade`.
- `height`: The height at which an upgrade should be executed and the node will be restarted.
- `info`: Can be empty. **Only needed for live networks where we want cosmovisor to upgrade nodes automatically**.

:::tip

When `cosmovisor` is configured to automatically download binaries, it will pull the binary from the link provided in this field and perform a hash verification (which is also optional). We only know the hashes **AFTER** the release has been cut and CI created artifacts for this version.

:::

### Validate the URLs (live network only)

The URLs of the binaries contain checksums. It is critical to ensure they are correct.
Otherwise Cosmovisor won't be able to download the binaries and go through the upgrade.

The command below (using toold build by the authors of Cosmosvisor) can be used to achieve the above:

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

## Submitting the upgrade on-chain

The `MsgSoftwareUpgrade` can be submitted using the following command:

```bash
poktrolld tx authz exec $PATH_TO_UPGRADE_TRANSACTION_JSON --from=pnf
```

If the transaction has been accepted, the upgrade plan can be viewed with this command:

```bash
poktrolld query upgrade plan
```

## Cancelling the upgrade plan

It is possible to cancel the upgrade before the upgrade plan height is reached. To do so, execute the following transaction:

```bash
poktrolld tx authz exec tools/scripts/upgrades/authz_cancel_upgrade_tx.json --gas=auto --from=pnf
```

## Testing the Upgrade

:::warning
Note that for local testing, `cosmovisor` won't pull the binary from the info field.
:::

### LocalNet

LocalNet **DOES NOT** support `cosmovisor` and automatic upgrades at the moment.

However, **IT IS NOT NEEDED** to simulate and test the upgrade procedure.

#### LocalNet Upgrade tl;dr

1. Pull git repo with old version (separate directory)
2. Download release binary of the old version
3. Wipe LocalNet data and generate genesis using OLD version
4. Start node using anOLD binary
5. Write and submit an upgrade transaction on-chain
6. When the Upgrade Plan height is reached, stop the old node and run the new binary
7. Observe the behavior

#### LocalNet Upgrade Full Example Walkthrough

Testing an upgrade requires a network running on an old version.

Ensure LocalNet is running using a binary from the [previous release you wish to upgrade **FROM**](https://github.com/pokt-network/poktroll/releases). We also want to provision the network using this version, which requires us to pull the specific git tag.

1. Make a note of the version you want to test an upgrade **FROM**. This will be the **OLD** version. For example, let's imagine we're upgrading from `v0.0.9`.
2. Pull a new `poktroll` repo (will be used as an "old" version):

   ```bash
   git clone https://github.com/pokt-network/poktroll.git poktroll-upgrade-old
   cd poktroll-upgrade-old
   git checkout v0.0.9

   # Download the v0.0.9 binary: https://github.com/pokt-network/poktroll/releases
   # CHANGE POKTROLLD_VERSION and ARCH
   curl -L "https://github.com/pokt-network/poktroll/releases/download/${POKTROLLD_VERSION}/poktroll_linux_${ARCH}.tar.gz" | tar -zxvf - -C .

   # Validate the version
   ./poktrolld version
   0.0.9
   ```

3. Stop LocalNet

   ```bash
   make localnet_down
   ```

4. Reset the data

   ```bash
   ./poktrolld comet unsafe-reset-all
   ```

5. Create new genesis using old version (from `poktroll-upgrade-old` dir)

   ```bash
   make localnet_regenesis
   ```

6. Start the network

   ```bash
   ./poktrolld start
   ```

7. [Write](#writing-an-upgrade-transaction) and [Submit](#submitting-the-upgrade-on-chain) a transaction. For example:

   ```bash
   poktrolld tx authz exec tools/scripts/upgrades/local_test_v0.0.9-2.json --from=pnf`
   ```

8. Verify the plan is active

   ```bash
   poktrolld query upgrade plan
   ```

9. Wait until the height is reached and the old node dies due to the error: `ERR UPGRADE "v0.0.9-2" NEEDED at height`, which is expected.
10. At this point, switch to the repo with the **NEW** version - the code you wish to upgrade the network **TO**.
11. In the **NEW VERSION GIT REPO** you can build binaries using `go_develop`, `ignite_release` and `ignite_release_extract_binaries` make targets.
12. Start the new version from the **NEW VERSION REPO**:

    ```bash
    ./release_binaries/poktroll_darwin_arm64 start
    ```

13. Observe the behavior. Your node should go through the upgrade process and start using the new version.

### DevNet

DevNets currently do not support `cosmovisor`.

We use Kubernetes to manage software versions, including validators. Introducing another component to manage versions would be complex, requiring a re-architecture of our current solution to accommodate this change.

### TestNet

We currently deploy TestNet validators using Kubernetes with helm charts, which prevents us from managing the validator with `cosmovisor`. We do not control what other TestNet participants are running. However, if participants have deployed their nodes using the [cosmovisor guide](../../operate/run_a_node/full_node_cosmovisor.md), their nodes will upgrade automatically.

Until we transition to [cosmos-operator](https://github.com/strangelove-ventures/cosmos-operator), which supports scheduled upgrades (although not fully automatic like `cosmovisor`), we need to manually manage the process:

1. Estimate when the upgrade height will be reached.
2. When validator node(s) stop due to an upgrade, manually perform an ArgoCD apply and clean up old resources.
3. Monitor validator node(s) as they start and begin producing blocks.

:::tip
If you are a member of Grove, you can find the instructions to access the infrastructure [here](https://www.notion.so/buildwithgrove/How-to-re-genesis-a-Shannon-TestNet-a6230dd8869149c3a4c21613e3cfad15?pvs=4).
:::

### Mainnet

The Mainnet upgrade process is to be determined. We aim to develop and implement improved tooling for this environment.
