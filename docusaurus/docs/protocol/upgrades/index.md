---
sidebar_position: 1
---

# Protocol upgrades

## Overview

The Pocket Network is constantly evolving with regular protocol upgrades. We have a process of submitting software upgrades
via a DAO process where validator nodes can have consensus-breaking changes and be automatically restarted when using [cosmovisor](../../operate/run_a_node/full_node_cosmovisor.md) (or manually if not using `cosmovisor`).

## What is a protocol upgrade?

## List of upgrades

While you can find a list of [poktroll releases](https://github.com/pokt-network/poktroll/releases) on our GitHub, we also maintain a [list of upgrades](./upgrade_list.md) in documentation. This list includes information whether there is a breaking change or not, and if the manual intervention is required from the operator.

## Upgrade types

### Planned/Unplanned upgrades

**Planned** upgrades are the ones the team has been working on for a while and that have been announced in advance. They usually include new features, improvements to existing functionalities or optimizations.

**Unplanned** upgrades can happen at any time without prior notice. These may be due to bugs, security issues, network congestion, etc. if there's no other way to mitigate the issue. Such upgrades **might** require manual intervention from users and/or validators due to a potential hard fork.

### Breaking/Non-breaking upgrades

**Breaking** changes are those that might affect existing APIs, State Machine logic, etc. They usually imply some form of migration process for the network participants. The protocol team strives to reduce the need in manual interventions.

**Non-breaking** changes do not have such implications and can be applied without much trouble to the current state of the system.

### Manual interventions

While the risk is low, it is always possible the blockchain might suffer some unplanned issues. When such situations arise,
and there is a need for forking the network (such as un-determenistinc chain state), we will issue an upgrade notice that requires manual intervention from users and/or validators to ensure the health of the network.
