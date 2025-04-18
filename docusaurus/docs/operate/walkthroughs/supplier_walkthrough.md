---
title: Supplier & RelayMiner (~30 min)
sidebar_position: 5
---

## Supplier & RelayMiner Walkthrough <!-- omit in toc -->

- [What is a RelayMiner](#what-is-a-relayminer)
- [RelayMiner Operation Requirements](#relayminer-operation-requirements)
- [Hardware requirements](#hardware-requirements)
- [Docker Compose Example](#docker-compose-example)
- [Kubernetes Example](#kubernetes-example)

## What is a RelayMiner

See the [RelayMiner](../../protocol/actors/relay_miner.md) documentation for more
information on what a RelayMiner is. This page aims to provide links and
details on how to deploy and operate it.

## RelayMiner Operation Requirements

A RelayMiner requires the following:

1. A staked onchain [Supplier](../../protocol/actors/supplier.md) to provide services.
2. A connection to a [Full Node](./full_node_docker.md) to interact with the blockchain.

:::tip
It is crucial to deploy a [Full Node](full_node_docker.md) prior to setting up a RelayMiner.
This ensures the necessary infrastructure for blockchain communication is in place.
:::

## Hardware requirements

Please see the [Hardware Requirements](../configs/hardware_requirements.md#relayminer) page.

## Docker Compose Example

Please refer to the `Deploying a RelayMiner` section in [Docker compose walkthrough](../../operate/walkthroughs/docker_compose_walkthrough.md) for detailed instructions
on how to deploy a `RelayMiner` using `docker-compose`.

_TODO_DOCUMENT: Move over the relevant information from the `poktroll-docker-compose-example` repository into the docs_

## Kubernetes Example

_TODO_DOCUMENT: Provide an example using [strangelove-ventures/cosmos-operator](https://github.com/strangelove-ventures/cosmos-operator)._
