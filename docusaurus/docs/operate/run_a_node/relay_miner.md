---
title: RelayMiner
sidebar_position: 4
---

# RelayMiner <!-- omit in toc -->

- [What is a RelayMiner](#what-is-a-relayminer)
- [RelayMiner Operation Requirements](#relayminer-operation-requirements)
- [Docker Compose Example](#docker-compose-example)
- [Kubernetes Example](#kubernetes-example)

## What is a RelayMiner

See the [RelayMiner](../../protocol/actors/appgate_server.md) documentation for more
information on what a RelayMiner is. This page aims to provide links and
details on how to deploy and operate it.

## RelayMiner Operation Requirements

A RelayMiner requires the following:

1. A staked on-chain [Supplier](../../protocol/actors/supplier.md) to provide services.
2. A connection to a [Full Node](./full_node.md) to interact with the blockchain.

:::tip
It is crucial to deploy a [Full Node](full_node.md) prior to setting up a RelayMiner.
This ensures the necessary infrastructure for blockchain communication is in place.
:::

## Docker Compose Example

Please refer to the `Deploying a RelayMiner` section in [poktroll-docker-compose-example](https://github.com/pokt-network/poktroll-docker-compose-example#deploying-a-relay-miner)
GitHub repository on how to deploy an AppGate Server using `docker-compose`.

_TODO: Move over the relevant information from the `poktroll-docker-compose-example` repository into the docs_

## Kubernetes Example

_TODO: Provide an example using [strangelove-ventures/cosmos-operator](https://github.com/strangelove-ventures/cosmos-operator)._
