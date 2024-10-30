---
title: AppGate Server
sidebar_position: 4
---

# Run an AppGate Server <!-- omit in toc -->

- [What is AppGate Server?](#what-is-appgate-server)
- [AppGate Server Operation Requirements](#appgate-server-operation-requirements)
- [Hardware requirements](#hardware-requirements)
- [Docker Compose Example](#docker-compose-example)
- [Kubernetes Example](#kubernetes-example)

## What is AppGate Server?

See the [AppGate Server](../../protocol/actors/appgate_server.md) documentation for more
information on what an AppGate Server is. This page aims to provide links and
details on how to deploy and operate it.

## AppGate Server Operation Requirements

An AppGate Server requires the following:

1. A staked on-chain [Application](../../protocol/actors/application.md) to pay for services.
2. An optional on-chain [Gateway](../../protocol/actors/gateway.md) to optionally proxy services.
3. A connection to a [Full Node](./full_node_docker.md) to interact with the blockchain.

:::tip
It is crucial to deploy a [Full Node](full_node_docker.md) prior to setting up a RelayMiner.
This ensures the necessary infrastructure for blockchain communication is in place.
:::

## Hardware requirements

Please see the [Hardware Requirements](./hardware_requirements.md#appgate-server--gateway) page.

## Docker Compose Example

Please refer to the `Deploying an AppGate Server` section in [poktroll-docker-compose-example](https://github.com/pokt-network/poktroll-docker-compose-example#deploying-an-appgate-server)
GitHub repository on how to deploy an AppGate Server using `docker-compose`.

## Kubernetes Example

_TODO_DOCUMENT: Provide an example using [strangelove-ventures/cosmos-operator](https://github.com/strangelove-ventures/cosmos-operator)._
