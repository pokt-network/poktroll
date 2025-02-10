---
sidebar_position: 7
title: Chain Halt Recovery
---

## Chain Halt Recovery <!-- omit in toc -->

This document describes how to recover from a chain halt.

It assumes that the cause of the chain halt has been identified, and that the
new release has been created and verified to function correctly.

:::tip

See [Chain Halt Troubleshooting](chain_halt_troubleshooting.md) for more information on identifying the cause of a chain halt.

:::

- [Background](#background)
- [Resolving halts during a network upgrade](#resolving-halts-during-a-network-upgrade)
  - [Manual binary replacement (preferred)](#manual-binary-replacement-preferred)
  - [Rollback, fork and upgrade](#rollback-fork-and-upgrade)
  - [Troubleshooting](#troubleshooting)
    - [Data rollback - retrieving snapshot at a specific height (step 5)](#data-rollback---retrieving-snapshot-at-a-specific-height-step-5)
    - [Validator Isolation - risks (step 6)](#validator-isolation---risks-step-6)

## Background

Pocket network is built on top of `cosmos-sdk`, which utilizes the CometBFT consensus engine.
Comet's Byzantine Fault Tolerant (BFT) consensus algorithm requires that **at least** 2/3 of Validators
are online and voting for the same block to reach a consensus. In order to maintain liveness
and avoid a chain-halt, we need the majority (> 2/3) of Validators to participate
and use the same version of the software.

## Resolving halts during a network upgrade

If the halt is caused by the network upgrade, it is possible the solution can be as simple as
skipping an upgrade (i.e. `unsafe-skip-upgrade`) and creating a new (fixed) upgrade.

Read more about [upgrade contingency plans](contigency_plans.md).

### Manual binary replacement (preferred)

:::note

This is the preferred way of resolving consensus-breaking issues.

**Significant side effect**: this breaks an ability to sync from genesis **without manual interventions**.
For example, when a consensus-breaking issue occurs on a node that is synching from the first block, node operators need
to manually replace the binary with the new one. There are efforts underway to mitigate this issue, including
configuration for `cosmovisor` that could automate the process.

<!-- TODO_MAINNET(@okdas): Add links to Cosmovisor documentation on how the new UX can be used to automate syncing from genesis without human input. -->

:::

Since the chain is not moving, **it is impossible** to issue an automatic upgrade with an upgrade plan. Instead,
we need **social consensus** to manually replace the binary and get the chain moving.

The steps to doing so are:

1. Prepare and verify a new binary that addresses the consensus-breaking issue.
2. Reach out to the community and validators so they can upgrade the binary manually.
3. Update [the documentation](upgrade_list.md) to include a range a height when the binary needs
   to be replaced.

:::warning

TODO_MAINNET(@okdas):

1. **For step 2**: Investigate if the CometBFT rounds/steps need to be aligned as in Morse chain halts. See [this ref](https://docs.cometbft.com/v1.0/spec/consensus/consensus).
2. **For step 3**: Add `cosmovisor` documentation so its configured to automatically replace the binary when synching from genesis.

:::

```mermaid
sequenceDiagram
    participant DevTeam
    participant Community
    participant Validators
    participant Documentation
    participant Network

    DevTeam->>DevTeam: 1. Prepare and verify new binary
    DevTeam->>Community: 2. Announce new binary and instructions
    DevTeam->>Validators: 2. Notify validators to upgrade manually
    Validators->>Validators: 2. Manually replace the binary
    Validators->>Network: 2. Restart nodes with new binary
    DevTeam->>Documentation: 3. Update documentation (GitHub Release and Upgrade List to include instructions)
    Validators-->>Network: Network resumes operation

```

### Rollback, fork and upgrade

:::info

These instructions are only relevant to Pocket Network's Shannon release.

We do not currently use `x/gov` or on-chain voting for upgrades.
Instead, all participants in our DAO vote on upgrades off-chain, and the Foundation
executes transactions on their behalf.

:::

:::warning

This should be avoided or more testing is required. In our tests, the full nodes were
propagating the existing blocks signed by the Validators, making it hard to rollback.

:::

**Performing a rollback is analogous to forking the network at the older height.**

However, if necessary, the instructions to follow are:

1. Prepare & verify a new binary that addresses the consensus-breaking issue.
2. [Create a release](release_process.md).
3. [Prepare an upgrade transaction](upgrade_procedure.md#writing-an-upgrade-transaction) to the new version.
4. Disconnect the `Validator set` from the rest of the network **3 blocks** prior to the height of the chain halt. For example:
   - Assume an issue at height `103`.
   - Revert the `validator set` to height `100`.
   - Submit an upgrade transaction at `101`.
   - Upgrade the chain at height `102`.
   - Avoid the issue at height `103`.
5. Ensure all validators rolled back to the same height and use the same snapshot ([how to get a snapshot](#data-rollback---retrieving-snapshot-at-a-specific-height-step-5))
   - The snapshot should be imported into each Validator's data directory.
   - This is necessary to ensure data continuity and prevent forks.
6. Isolate the `validator set` from full nodes - ([why this is necessary](#validator-isolation---risks-step-6)).
   - This is necessary to avoid full nodes from gossiping blocks that have been rolled back.
   - This may require using a firewall or a private network.
   - Validators should only be permitted to gossip blocks amongst themselves.
7. Start the `validator set` and perform the upgrade. For example, reiterating the process above:
   - Start all Validators at height `100`.
   - On block `101`, submit the `MsgSoftwareUpgrade` transaction with a `Plan.height` set to `102`.
   - `x/upgrade` will perform the upgrade in the `EndBlocker` of block `102`.
   - The node will stop climbing with an error waiting for the upgrade to be performed.
     - Cosmovisor deployments automatically replace the binary.
     - Manual deployments will require a manual replacement at this point.
   - Start the node back up.
8. Wait for the network to reach the height of the previous ledger (`104`+).
9. Allow validators to open their network to full nodes again.
   - **Note**: full nodes will need to perform the rollback or use a snapshot as well.

```mermaid
sequenceDiagram
    participant DevTeam
    participant Foundation
    participant Validators
    participant FullNodes
    %% participant Network

    DevTeam->>DevTeam: 1. Prepare & verify new binary
    DevTeam->>DevTeam: 2 & 3. Create a release & prepare upgrade transaction
    Validators->>Validators: 4 & 5. Roll back to height before issue or import snapshot
    Validators->>Validators: 6. Isolate from Full Nodes
    Foundation->>Validators: 7. Distribute upgrade transaction
    Validators->>Validators: 7. Start network and perform upgrade

    break
    Validators->>Validators: 8. Wait until previously problematic height elapses
    end

    Validators-->FullNodes: 9. Open network connections
    FullNodes-->>Validators: 9. Sync with updated network
    note over Validators,FullNodes: Network resumes operation
```

### Troubleshooting

#### Data rollback - retrieving snapshot at a specific height (step 5)

There are two ways to get a snapshot from a prior height:

1. Execute

   ```bash
   poktrolld rollback --hard
   ```

   repeately, until the command responds with the desired block number.

2. Use a snapshot from below the halt height (e.g. `100`) and start the node with `--halt-height=100` parameter so it only syncs up to certain height and then
   gracefully shuts down. Add this argument to `poktrolld start` like this:

   ```bash
   poktrolld start --halt-height=100
   ```

#### Validator Isolation - risks (step 6)

Having at least one node that has knowledge of the forking ledger can jeopardize the whole process. In particular, the
following errors in logs are the sign of the nodes syncing blocks from the wrong fork:

- `found conflicting vote from ourselves; did you unsafe_reset a validator?`
- `conflicting votes from validator`
