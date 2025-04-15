---
title: Introduction to Protocol Upgrades
sidebar_position: 1
---

:::info GitHub Release vs Protocol Upgrade

Not every [GitHub release](https://github.com/pokt-network/poktroll/releases) will become a protocol upgrade.

:::

Pocket Network is continuously evolving through regular protocol upgrades.

We support software upgrades via an off-chain DAO process, allowing validator nodes
to incorporate `consensus-breaking` changes. Upgrades are executed by PNF on behalf of the DAO.

These upgrades can be automatically applied when using [Cosmovisor](../walkthroughs/full_node_walkthrough.md),
or manually if not using `cosmovisor`.

## Table of Contents <!-- omit in toc -->

- [What is a Protocol Upgrade?](#what-is-a-protocol-upgrade)
- [List of Upgrades](#list-of-upgrades)
- [When is an Protocol Upgrade Warranted?](#when-is-an-protocol-upgrade-warranted)
- [Protocol \& Software Process Overview](#protocol--software-process-overview)
- [Upgrade Types](#upgrade-types)
  - [Planned vs. Unplanned Upgrades](#planned-vs-unplanned-upgrades)
  - [Breaking vs. Non-breaking Upgrades](#breaking-vs-non-breaking-upgrades)
  - [Manual Interventions](#manual-interventions)

## What is a Protocol Upgrade?

A protocol upgrade is a process of updating Pocket Network on-chain software to
introduce new features, improve existing functionalities, or address critical issues.

These upgrades ensure the network remains secure, efficient, and up-to-date with the latest technological advancements.

## List of Upgrades

While you can find a comprehensive list of [pocket releases](https://github.com/pokt-network/poktroll/releases) on our GitHub, we also maintain a [list of upgrades](4_upgrade_list.md) in our documentation. This list provides valuable information, including whether an upgrade involves breaking changes and if manual intervention is required from operators.

## When is an Protocol Upgrade Warranted?

:::warning TODO

TODO(@red-0ne): Document what a `state breaking` change is.

:::

There are three types of update

1. **`consensus-breaking` changes**: Protocol Upgrade & GitHub Release Required. _For example, changes to protobufs._
2. **Node Software changes**: GitHub Release Required and Protocol Upgrade Optional but Highly Recommended. _For examples, performance improvements._
3. **Software Release**: GitHub Release Required and no Protocol Upgrade Required. _For example, new utilities in the CLI._

**Identify consensus breaking changes** by:

1. Reviewing merged [Pull Requests (PRs) with the `consensus-breaking` label](https://github.com/pokt-network/poktroll/issues?q=label%3Aconsensus-breaking+) since the last release. It is not a source of truth, but directionality correct.
2. Looking for breaking changes in `.proto` files
3. Looking for breaking changes in the `x/` directories
4. Identifying new onchain parameters or authorizations

:::info Non-exhaustive list

Note that the above is a non-exhaustive list and requires protocol expertise to identify all potential `consensus-breaking` changes.
:::

## Protocol & Software Process Overview

When a `consensus-breaking` change is made to the protocol, we must carefully evaluate and implement an upgrade path that
allows existing nodes to transition safely from one software version to another without disruption.

This process involves several key steps:

1. **Proposal**: The DAO drafts an upgrade proposal using our offchain governance system.
2. **Implementation**: The proposed changes are implemented in the codebase.
3. **Testing**: Thorough testing of the proposed changes is conducted in devnet and testnet environments before mainnet deployment.
4. **Announcement**: Upon successful testing, we announce the upgrade through our social media channels and community forums.
5. **Deployment**: An upgrade transaction is sent to the network, allowing node operators using [Cosmovisor](../walkthroughs/full_node_walkthrough.md) to automatically upgrade their nodes at the specified block height.
6. **Monitoring**: Post-deployment, we closely monitor the network to ensure everything functions as expected.

## Upgrade Types

### Planned vs. Unplanned Upgrades

**Planned upgrades** are those that our team has been developing for some time and have been announced in advance.
These typically include new features, improvements to existing functionalities, or optimizations.

**Unplanned upgrades** may occur at any time without prior notice.
These are often necessary due to unforeseen circumstances such as bugs, security issues, chain halts, or network congestion when no other mitigation is possible.
Such upgrades may require manual intervention from users and/or validators, potentially resulting in a hard fork.

### Breaking vs. Non-breaking Upgrades

**Breaking changes** are those that may affect existing APIs, State Machine logic, or other critical components.
They usually require some form of migration process for network participants.
Our protocol team strives to minimize the need for manual interventions in these cases.

**Non-breaking changes** do not have such implications and can be applied without significant disruption to the current state of the system.

### Manual Interventions

While the risk is low, it's possible that the blockchain may encounter unexpected issues.
In situations where forking the network becomes necessary (such as in cases of non-deterministic chain state), we will issue an upgrade notice requiring manual intervention from users and/or validators to ensure the network's health and integrity.
