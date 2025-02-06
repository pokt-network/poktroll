---
title: Gateway Walkthrough
sidebar_position: 6
---

# Run a Gateway <!-- omit in toc -->

- [What is PATH Gateway?](#what-is-path-gateway)
- [PATH Gateway Operation Requirements](#path-gateway-operation-requirements)
- [Hardware requirements](#hardware-requirements)
- [Docker Compose Example](#docker-compose-example)
- [Kubernetes Example](#kubernetes-example)

## What is PATH Gateway?

See the [PATH Gateway](https://path.grove.city) documentation for more
information on what a `PATH Gateway` is. This page aims to provide links and
details on how to deploy and operate it.

## PATH Gateway Operation Requirements

A PATH Gateway requires the following:

1. A staked onchain [Application](../../protocol/actors/application.md) to pay for services.
2. An optional onchain [Gateway](../../protocol/actors/gateway.md) to optionally proxy services.
3. A connection to a [Full Node](./full_node_docker.md) to interact with the blockchain.

:::tip
It is crucial to deploy a [Full Node](full_node_docker.md) prior to setting up a RelayMiner.
This ensures the necessary infrastructure for blockchain communication is in place.
:::

## Hardware requirements

Please see the [Hardware Requirements](../configs/hardware_requirements.md#path-gateway) page.

## Docker Compose Example

Please refer to the `Deploying a PATH Gateway` section in [Docker compose walkthrough](../quickstart/docker_compose_walkthrough#d-creating-a-gateway-deploying-a-path-gateway)
on how to deploy a `PATH Gateway` using `docker-compose`.

## Kubernetes Example

_TODO_DOCUMENT: Provide an example using [strangelove-ventures/cosmos-operator](https://github.com/strangelove-ventures/cosmos-operator)._
