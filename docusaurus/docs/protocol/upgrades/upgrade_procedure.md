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
- [Submitting the upgrade onchain](#submitting-the-upgrade-onchain)
- [Testing the Upgrade](#testing-the-upgrade)
  - [LocalNet](#localnet)
  - [DevNet](#devnet)
  - [TestNet](#testnet)
  - [Mainnet](#mainnet)

## Overview <!-- omit in toc -->

When a consensus-breaking change is made to the protocol, we must carefully evaluate and implement an upgrade path that allows existing nodes to transition safely from one software version to another without disruption. This process involves several key steps:

1. **Proposal**: The DAO drafts an upgrade proposal using our offchain governance system.
2. **Implementation**: The proposed changes are implemented in the codebase.
3. **Testing**: Thorough testing of the proposed changes is conducted in devnet and testnet environments before mainnet deployment.
4. **Announcement**: Upon successful testing, we announce the upgrade through our social media channels and community forums.
5. **Deployment**: An upgrade transaction is sent to the network, allowing node operators using [Cosmovisor](../../operate/run_a_node/full_node_walkthrough.md) to automatically upgrade their nodes at the specified block height.
6. **Monitoring**: Post-deployment, we closely monitor the network to ensure everything functions as expected.

## When is an Upgrade Warranted?

An upgrade is necessary whenever there's an API, State Machine, or other Consensus breaking change in the version we're about to release.

## Implementing the Upgrade

1. When a new version includes a consensus-breaking change, plan for the next protocol upgrade:
   - If there's a change to a specific module, bump that module's consensus version.
   - Note any potential parameter changes to include in the upgrade.
2. Create a new upgrade in `app/upgrades`:
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
- `info`: While this field can theoretically contain any information about the upgrade, in practice, `cosmovisor`uses it to obtain information about the binaries. When`cosmovisor` is configured to automatically download binaries, it will pull the binary from the link provided in this field and perform a hash verification (which is optional).

## Submitting the upgrade onchain

The `MsgSoftwareUpgrade` can be submitted using the following command:

```bash
poktrolld tx authz exec PATH_TO_TRANSACTION_JSON --from pnf
```

If the transaction has been accepted, upgrade plan can be viewed with this command:

```bash
poktrolld query upgrade plan
```

## Testing the Upgrade

:::warning
Note that for local testing, `cosmovisor` won't pull the binary from the info field.
:::

### LocalNet

LocalNet currently does not support `cosmovisor` and automatic upgrades. However, we have provided scripts to facilitate local testing in the `tools/scripts/upgrades` directory:

1. Modify `tools/scripts/upgrades/authz_upgrade_tx_example_v0.0.4_height_30.json` to reflect the name of the upgrade and the height at which it should be scheduled.

2. Check and update the `tools/scripts/upgrades/cosmovisor-start-node.sh` to point to the correct binaries:

   - The old binary should be compiled to work before the upgrade.
   - The new binary should contain the upgrade logic to be executed immediately after the node is started using the new binary.

3. Run `bash tools/scripts/upgrades/cosmovisor-start-node.sh` to wipe the `~/.poktroll` directory and place binaries in the correct locations.

4. Execute the transaction as shown in [Submitting the upgrade onchain](#submitting-the-upgrade-onchain) section above.

### DevNet

DevNets currently do not support `cosmovisor`.

We use Kubernetes to manage software versions, including validators. Introducing another component to manage versions would be complex, requiring a re-architecture of our current solution to accommodate this change.

### TestNet

We currently deploy TestNet validators using Kubernetes with helm charts, which prevents us from managing the validator with `cosmovisor`. We do not control what other TestNet participants are running. However, if participants have deployed their nodes using the [cosmovisor guide](../../operate/run_a_node/full_node_walkthrough.md), their nodes will upgrade automatically.

Until we transition to [cosmos-operator](https://github.com/strangelove-ventures/cosmos-operator), which supports scheduled upgrades (although not fully automatic like `cosmovisor`), we need to manually manage the process:

1. Estimate when the upgrade height will be reached.
2. When validator node(s) stop due to an upgrade, manually perform an ArgoCD apply and clean up old resources.
3. Monitor validator node(s) as they start and begin producing blocks.

:::tip
If you are a member of Grove, you can find the instructions to access the infrastructure [here](https://www.notion.so/buildwithgrove/How-to-re-genesis-a-Shannon-TestNet-a6230dd8869149c3a4c21613e3cfad15?pvs=4).
:::

### Mainnet

The Mainnet upgrade process is to be determined. We aim to develop and implement improved tooling for this environment.
