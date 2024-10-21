---
sidebar_position: 7
title: Chain Halt Recovery
---

## Chain Halt Recovery <!-- omit in toc -->

This document describes how to recover from a chain halt. It assumes the cause of
the chain halt has been identified, the new release has been created, and verified
function correctly.

:::tip
See [Chain Halt Troubleshooting](./chain_halt_troubleshooting.md) for more information on identifying the cause of a chain halt.
:::

- [Background](#background)
- [Resolving halts during a network upgrade](#resolving-halts-during-a-network-upgrade)
  - [Manual binary replacement (preferred)](#manual-binary-replacement-preferred)
  - [Rollback, fork and upgrade](#rollback-fork-and-upgrade)

## Background

Pocket network is built on top of `cosmos-sdk`, which utilizes the CometBFT consensus engine.
Byzantine Fault Tolerant (BFT) consensus algorithm requires that **at least** 2/3 of Validators
are online and voting for the same block to reach a consensus. In order to maintain liveness
and avoid a chain-halt, we need the majority (> 2/3) of Validators to participate
and use the same version of the software.

## Resolving halts during a network upgrade

If the halt is caused by the network upgrade, it is possible the solution can be as simple as
skipping an upgrade (i.e. `unsafe-skip-upgrade`) and creating a new (fixed) upgrade.

Read more about [upgrade contingency plans](../../protocol/upgrades/contigency_plans.md).

### Manual binary replacement (preferred)

:::note

**This is preferred way of resolving the consensus-breaking issues**.

:::

Since the chain is not moving, **it is impossible** to issue an automatic upgrade with an upgrade plan.

Instead, we need **social consensus** to manually replace the binary and get the chain moving.

Currently this involves synching the network from genesis breaking a way to sync the network from genesis without human interaction, but there are some plans to make the process less painful in the future.

<!-- TODO_IMPROVE(@okdas): add links to Cosmovisor documentation how the new UX can be used to automate syncing from genesis without human input. -->

### Rollback, fork and upgrade

:::info

This part is relevant for Pocket Network Shannon release only, as we do not rely on `x/gov` module for upgrades in Shannon. Instead, our DAO can issue upgrade transactions on the Pocket Network chain directly. Conventional `cosmos-sdk` upgrade process would require to go through the voting process to issue an upgrade.

:::

Performing a rollback basically means forking the network at the older height. Modern CometBFT versions are incredibly hard to fork. As a result, **it is not recommended to perform rollbacks** unless absolutely necessary. If we do decide to go ahead with a rollback, these are the steps:

- Prepare and verify the new version that addresses the consensus-breaking issue.
- [Create a release](../../protocol/upgrades/release_process.md).
- [Prepare an upgrade transaction](../../protocol/upgrades/upgrade_procedure.md#writing-an-upgrade-transaction) to the new version.
- Get the state of the validators on the network to **three blocks** prior to the consensus-breaking issue.
  - For example, if there was an issue at height `103`, we need to get the state to the height of `100`. At `101` we will submit an upgrade transaction so the chain upgrades on `102` and avoids the issue at height `103`.
  - Can be done in two ways:
    - `poktrolld rollback --hard` until the command responds with the desired block number. **OR,**
    - The node can be restored from the snapshot and started with `--halt-height=100` parameter so it only syncs up to certain height and then gracefully shuts down.
- **Make sure all validators use the same data directory** or have been rolled back to the same height.
- **Isolate validators from the other nodes** that have not been rolled back to the older state. If that means using a firewall or isolating from the internet - this is the way. Validators should be able to only gossip blocks between themselves. **Having at least one node that has knowledge of the forking ledger can jeopardize the whole process**. In particular, the following errors are the sign of the nodes populating existing blocks:
  - `found conflicting vote from ourselves; did you unsafe_reset a validator?`
  - `conflicting votes from validator`
- Start the network and perform an upgrade (following the example above):
  - We would not be able to submit a transaction at `100` (this needs to be investigated, but for some reason we were not able to) due to `signature verification failed; please verify account number (0) and chain-id  (poktroll): (unable to verify single signer signature): unauthorized`.
  - On block `101`, we will submit the `MsgSoftwareUpgrade` transaction with a `Plan.height` set to `102`.
  - `x/upgrade` performs an upgrade in the `EndBlocker` of the block `102` and waits for the node operator or `cosmovisor` to replace the binary.
- The network should go through successful upgrade and climb to the next block.
- After the chain has been reached over the height of the previous ledger (`104`+), validators can open the gates for other full nodes to join the network again. Full nodes can perform the rollback or use a snapshot as well.
