---
title: Protocol Upgrades Overview
sidebar_position: 1
---

:::important Managing Protocol Upgrades
This is a meta introduction to protocol upgrades.

**❗ If you need to trigger a protocol upgrade, see the [Protocol Upgrade Release Procedure](2_release_procedure.md) section ❗**
:::

Pocket Network evolves through protocol upgrades that keep the network secure, efficient, and up-to-date. These upgrades are coordinated by the DAO (offchain governance) and executed by validators (onchain).

Operators can apply upgrades automatically with [Cosmovisor](../../1_operate/2_walkthroughs/1_full_node_binary.md) or manually otherwise.

## Table of Contents <!-- omit in toc -->

- [What is a Protocol Upgrade?](#what-is-a-protocol-upgrade)
- [Where to Find Upgrade Info](#where-to-find-upgrade-info)
- [When is a Protocol Upgrade Needed?](#when-is-a-protocol-upgrade-needed)
- [Types of Upgrades](#types-of-upgrades)
  - [Consensus-Breaking vs. Non-breaking](#consensus-breaking-vs-non-breaking)
  - [Planned vs. Unplanned](#planned-vs-unplanned)
  - [Manual Interventions](#manual-interventions)
- [Identifying Consensus-Breaking Changes](#identifying-consensus-breaking-changes)
- [High Level Protocol \& Software Process](#high-level-protocol--software-process)

## What is a Protocol Upgrade?

A protocol upgrade changes the onchain software to:

- Add new features
- Improve or fix existing functionality
- Address critical issues

## Where to Find Upgrade Info

- [GitHub Releases](https://github.com/pokt-network/poktroll/releases): All software updates and release notes
- [Upgrade List](4_upgrade_list.md): Details on each upgrade, breaking changes, and manual steps

:::info Not All Releases Are Upgrades
Not every [GitHub release](https://github.com/pokt-network/poktroll/releases) triggers a protocol upgrade.
:::

## When is a Protocol Upgrade Needed?

| Update Type                    | Protocol Upgrade | GitHub Release | Consensus-Breaking | Example                                    |
| ------------------------------ | :--------------: | :------------: | :----------------: | ------------------------------------------ |
| **Consensus-breaking changes** |       Yes        |      Yes       |         ✅         | Changes to business logic in state machine |
| **State-breaking changes**     |       Yes        |      Yes       |         ✅         | Changes to protobufs/onchain state         |
| **Node (onchain) release**     |     Optional     |      Yes       |         ❌         | Performance improvements                   |
| **Offchain software release**  |        No        |      Yes       |         ❌         | New CLI utilities                          |

:::info State vs Consensus Breaking
All `state-breaking` changes are `consensus-breaking`, but not all `consensus-breaking` changes are `state-breaking`.
:::

## Types of Upgrades

### Consensus-Breaking vs. Non-breaking

- **Consensus-breaking:** All validators must upgrade to avoid chain splits.
- **Non-breaking:** Backward compatible; no immediate action required.

### Planned vs. Unplanned

- **Planned:** Scheduled and announced (features, improvements, maintenance)
- **Unplanned:** Urgent, for bugs/security/chain halts; may require manual steps and can cause a hard fork.

### Manual Interventions

- Some upgrades need manual steps from node operators or validators.
- Always review upgrade notes for manual intervention requirements.

## Identifying Consensus-Breaking Changes

To spot `consensus-breaking` changes, check:

1. [PRs with the `consensus-breaking` label](https://github.com/pokt-network/poktroll/issues?q=label%3Aconsensus-breaking+) since the last release
2. `.proto` files for breaking protobuf changes
3. `x/` directories for breaking source code changes
4. New onchain parameters or authorizations

If a network fork is needed (e.g., non-deterministic state), an upgrade notice will be issued requiring manual intervention by users/validators to protect network integrity.

:::info Not Exhaustive
This list is not exhaustive; protocol expertise is required to identify all possible `consensus-breaking` changes.
:::

## High Level Protocol & Software Process

For any `consensus-breaking` change, upgrades follow this path:

1. **Proposal:** DAO drafts and discusses the upgrade offchain.
2. **Implementation:** Changes are made in the codebase.
3. **Testing:** All changes are tested in devnet/testnet before mainnet.
4. **Announcement:** Upgrade details are shared with the community.
5. **Deployment:** Upgrade transaction is sent; Cosmovisor users upgrade automatically at the set block height.
6. **Monitoring:** Network is monitored post-upgrade for issues.
