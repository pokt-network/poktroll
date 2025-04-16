---
title: Introduction to Protocol Upgrades
sidebar_position: 1
---

:::info GitHub Release vs Protocol Upgrade

Not every [GitHub release](https://github.com/pokt-network/poktroll/releases) will become a protocol upgrade.

:::

Pocket Network is continuously evolving through regular protocol upgrades.

The DAO leads offchain governance and comes to agreement on upgrades through social consensus.

Validators support onchain `consensus-breaking` changes that were agreed on by the DAO offchain and triggered by PNF onchain. These upgrades can be automatically applied when using [Cosmovisor](../walkthroughs/full_node_walkthrough.md), or manually if not using `cosmovisor`.

## Table of Contents <!-- omit in toc -->

- [What is a Protocol Upgrade?](#what-is-a-protocol-upgrade)
- [Where to Find Upgrade Info](#where-to-find-upgrade-info)
- [When is a Protocol Upgrade Needed?](#when-is-a-protocol-upgrade-needed)
- [Protocol \& Software Process Overview](#protocol--software-process-overview)
- [Types of Upgrades](#types-of-upgrades)
  - [Planned vs. Unplanned](#planned-vs-unplanned)
  - [Breaking vs. Non-breaking](#breaking-vs-non-breaking)
  - [Manual Interventions](#manual-interventions)
- [Identifying Consensus-Breaking Changes](#identifying-consensus-breaking-changes)

## What is a Protocol Upgrade?

A protocol upgrade updates Pocket Network's onchain software to:

- Add new features
- Improve existing functionality
- Fix critical issues

These keep the network secure, efficient, and up-to-date.

## Where to Find Upgrade Info

- [GitHub Releases](https://github.com/pokt-network/poktroll/releases): Artifacts and release notes for every software update
- [Upgrade List](4_upgrade_list.md): Info on each upgrade, including breaking changes and manual intervention requirements

## When is a Protocol Upgrade Needed?

There are three types of updates:

1. **Consensus-breaking changes**
   - Protocol upgrade & GitHub release required
   - Example: changes to protobufs affecting core tokenomic business logic
2. **Node Software changes**
   - Protocol upgrade optional (but highly recommended); GitHub release required
   - Example: performance improvements that don't affect consensus
3. **Software Release**
   - Protocol upgrade NOT needed; GitHub release only
   - Example: new CLI utilities

## Protocol & Software Process Overview

When a `consensus-breaking` change is made to the protocol, we must carefully evaluate and implement an upgrade path that allows existing nodes to transition safely from one software version to another without disruption.

This process involves several key steps:

1. **Proposal**: The DAO drafts an upgrade proposal using our offchain governance system.
2. **Implementation**: The proposed changes are implemented in the codebase.
3. **Testing**: Thorough testing of the proposed changes is conducted in devnet and testnet environments before mainnet deployment.
4. **Announcement**: Upon successful testing, we announce the upgrade through our social media channels and community forums.
5. **Deployment**: An upgrade transaction is sent to the network, allowing node operators using [Cosmovisor](../walkthroughs/full_node_walkthrough.md) to automatically upgrade their nodes at the specified block height.
6. **Monitoring**: Post-deployment, we closely monitor the network to ensure everything functions as expected.

## Types of Upgrades

### Planned vs. Unplanned

- **Planned:** Scheduled, communicated in advance (features, improvements, maintenance)
- **Unplanned:** Urgent, in response to bugs/security issues/chain halts/network congestion when no other mitigation is possible. May require manual intervention and can result in a hard fork.

### Breaking vs. Non-breaking

- **Breaking:** All validators must upgrade to maintain consensus. Not upgrading may cause a chain split.
- **Non-breaking:** Backward compatible. No immediate validator action required.

### Manual Interventions

- Some upgrades require manual steps from node operators/validators.
- Always check upgrade notes for manual intervention requirements.

## Identifying Consensus-Breaking Changes

To identify `consensus-breaking` changes, review:

1. `consensus-breaking` label - Reviewing merged [Pull Requests (PRs) with the `consensus-breaking` label](https://github.com/pokt-network/poktroll/issues?q=label%3Aconsensus-breaking+) since the last release.
2. `.proto` files - Looking for breaking changes in protobufs
3. `x/` directories - Looking for breaking changes in the source code
4. `Parameters` - Identify new onchain parameters or authorizations

:::info Non-exhaustive list

Note that the above is a non-exhaustive list and requires protocol expertise to identify all potential `consensus-breaking` changes.
:::
In situations where forking the network becomes necessary (such as in cases of non-deterministic chain state), we will issue an upgrade notice requiring manual intervention from users and/or validators to ensure the network's health and integrity.
