---
title: Full Node (Docker)
sidebar_position: 3
---

# Run a Full Node using Docker <!-- omit in toc -->

- [What is a Full Node](#what-is-a-full-node)
- [Roles \& Responsibilities](#roles--responsibilities)
- [Types of Full Nodes](#types-of-full-nodes)
- [Pocket Network Full Nodes](#pocket-network-full-nodes)
- [Hardware requirements](#hardware-requirements)
- [Docker Compose Example](#docker-compose-example)
- [Kubernetes Example](#kubernetes-example)

## What is a Full Node

In blockchain networks, a Full Node retains a complete copy of the ledger.

You can visit the [Cosmos SDK documentation](https://docs.cosmos.network/main/user/run-node/run-node)
for more information on Full Nodes.

## Roles & Responsibilities

It is usually responsible for:

1. Verifying all committed transactions and blocks
2. Increase network security through data redundancy
3. Fostering decentralization
4. Gossiping blocks & transactions to other nodes

It is not responsible for:

1. Proposing new blocks
2. Participating in consensus

## Types of Full Nodes

There are two types of Full Nodes:

1. **Archive Nodes**: These nodes store the entire history of the blockchain.
2. **Pruning Nodes**: These nodes store only the most recent blocks and transactions.

## Pocket Network Full Nodes

Within Pocket Network, the role of Full Nodes is pivotal for Node Runners. These
nodes needed for off-chain entities like [RelayMiners](./relay_miner.md) and
[AppGates](./appgate_server.md), which rely on interaction with the Pocket Network
blockchain for full functionality.

This guide outlines how to configure, deploy and maintain Full Nodes.

## Hardware requirements

Please see the [Hardware Requirements](./hardware_requirements.md#validator--full-node) page.

## Docker Compose Example

Please refer to the `Deploying a Full Node` section in [Docker Compose Walkthrough](../quickstart/docker_compose_walkthrough.md)
on how to deploy a Full Node using `docker-compose`.

## Kubernetes Example

_TODO_DOCUMENT: Provide an example using [strangelove-ventures/cosmos-operator](https://github.com/strangelove-ventures/cosmos-operator)._
