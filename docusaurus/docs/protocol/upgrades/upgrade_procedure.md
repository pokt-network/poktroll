---
title: Upgrade procedure
sidebar_position: 2
---

# Upgrade procedure <!-- omit in toc -->

:::warning

This page describes protocol upgrades - an internal to the protocol team process. If you're interested in upgading your
Pocket Network node, check our [releases page](https://github.com/pokt-network/poktroll/releases) for upgrade
instructions and changelogs.

:::

- [Overview](#overview)
- [When is upgrade warranted?](#when-is-upgrade-warranted)
- [Implementing the upgrade](#implementing-the-upgrade)
- [Writing an upgrade transaction](#writing-an-upgrade-transaction)
- [Testing the upgrade](#testing-the-upgrade)
  - [LocalNet](#localnet)
  - [DevNet](#devnet)
  - [TestNet](#testnet)
  - [Mainnet](#mainnet)


## Overview

Whenever a consensus-breaking change is made to the protocol, we need to carefully evaluate and implement an upgrade path that allows existing nodes to safely transition from one version of the software to another without disruption. This process involves several steps:

- **Proposal**: A proposal for the upgrade is drafted by the DAO. We have an off-chain government proposal system, and do not enable a governance process on-chain.
- **Implementation**: The proposed changes are implemented in the codebase.
- **Testing**: The proposed changes are thoroughly tested in devnet and testnet environment before being deployed to mainnet.
- **Announcement**: Once testing is complete and no issues are found, an announcement is made on our social media channels and through our community forums.
- **Deployment**: The upgrade transaction is sent to the network. This allows node operators running [cosmovisor](../../operate/run_a_node/full_node_cosmovisor.md) to automatically upgrade their nodes at the previously specified block height.
- **Monitoring**: After deployment, we monitor the network to ensure that everything is functioning as expected.

## When is upgrade warranted?

We have to go trough an upgrade whenever there is an API/State Machine/Consensus breaking change in the version we are
about to release.

:::info

TODO_DISCUSS: on-chain upgrades can be also beneficial to automate the binary download/upgrade the node operators even when there
are not consensus-breaking changes. Do we want to support that use-case?

:::

## Implementing the upgrade

1. Once it is determined the new version includes a consensus-breaking change, we should plan for the next protocol upgrade.
   1. If there's a chage to the specific module, we should bump the consensus version of that module.
   2. Make a note of any potential parameter changes we need to do to include in the upgrade.
2. Create new upgrade in `app/upgrades`.
   1. Check `historical.go` for past upgrades and examples.
   2. Cosmos-sdk has some useful documentation pages on upgrades [here](https://docs.cosmos.network/main/build/building-apps/app-upgrade) and [here](https://docs.cosmos.network/main/build/modules/upgrade).

## Writing an upgrade transaction

When the upgrade is implemented in code, we can schedule it on network. An upgrade transaction includes a [Plan](https://github.com/cosmos/cosmos-sdk/blob/0fda53f265de4bcf4be1a13ea9fad450fc2e66d4/x/upgrade/proto/cosmos/upgrade/v1beta1/upgrade.proto#L14). A typical upgrade transaction will look like the following:

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
          "info": "{\"binaries\":{\"linux\/amd64\":\"https:\/\/github.com\/pokt-network\/poktroll\/releases\/download\/v0.0.4\/poktroll_linux_amd64.tar.gz?checksum=sha256:49d2bcea02702f3dcb082054dc4e7fdd93c89fcd6ff04f2bf50227dacc455638\",\"linux\/arm64\":\"https:\/\/github.com\/pokt-network\/poktroll\/releases\/download\/v0.0.4\/poktroll_linux_arm64.tar.gz?checksum=sha256:698f3fa8fa577795e330763f1dbb89a8081b552724aa154f5029d16a34baa7d8\",\"darwin\/amd64\":\"https:\/\/github.com\/pokt-network\/poktroll\/releases\/download\/v0.0.4\/poktroll_darwin_amd64.tar.gz?checksum=sha256:5ecb351fb2f1fc06013e328e5c0f245ac5e815c0b82fb6ceed61bc71b18bf8e9\",\"darwin\/arm64\":\"https:\/\/github.com\/pokt-network\/poktroll\/releases\/download\/v0.0.4\/poktroll_darwin_arm64.tar.gz?checksum=sha256:a935ab83cd770880b62d6aded3fc8dd37a30bfd15b30022e473e8387304e1c70\"}}"
        }
      }
    ]
  }
}
```

**name**: Name of the upgrade. It should match the `VersionName` of `upgrades.Upgrade`.
**height**: The height at which an upgrade should be executed and the node will be restarted.
**info**: Theoretically, it can contain any information about the upgrade. In practice, `cosmovisor` gets information about the binaries from this field. When `cosmovisor` is configured to automatically download binaries, it will pull the binary from the link and perform a hash verification (optional).

## Testing the upgrade

### LocalNet

LocalNet does not currently support `cosmovisor` and automatic upgrades. We have some scripts that can facilitate
local testing in `tools/scripts/upgrades` directory:

1. Modify `tools/scripts/upgrades/authz_upgrade_tx.json` to reflect a name of the upgrade and a height it should be scheduled for. As we're testing locally, cosmovisor won't pull the binary from the `info` field.
2. Modify `tools/scripts/upgrades/start-node.sh` to point to the correct binaries: old binary is compiled to work before the upgrade, and the new binary should contain the upgrade logic that will be executed right after the node is started using new binary.
3. Run `bash tools/scripts/upgrades/start-node.sh` that will wipe `~/.poktroll` directory and place binaries in the correct locations.
4. Run `bash tools/scripts/upgrades/submit-upgrade.sh` that will schedule the upgrade (using `authz_upgrade_tx.json`) and show the information about scheduled upgrade.

### DevNet

DevNets do not currently support `cosmovisor`.

We use Kubernetes to manage software versions we run, including validators. Introducing another piece that will manage
versions for us will be tricky and we need to re-architect the current solution to accomodate this.

### TestNet

We currently deploy TestNet validator using Kubernetes using helm charts, which currently prevents us from managing the
validator with `cosmovisor`. We do not control what other participants of TestNet are running. If they have their nodes
deployed using [cosmovisor guide](../../operate/run_a_node/full_node_cosmovisor.md) they will get upgraded automatically.

Until we switch to [cosmos-operator](https://github.com/strangelove-ventures/cosmos-operator?tab=readme-ov-file), which
supports scheduled (although not fully automatic like `cosmovisor` has) upgrades, we can "babysit" the process. We'd
need to:

1. Time when the hight of an upgrade will be reached.
2. When the validator node(s) stop due to an upgrade, perform an ArgoCD apply and old resource cleanup manually.
3. Monitor validator node(s) start and produce blocks.

### Mainnet

TBD, but we should have better tooling then.