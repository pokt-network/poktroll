---
title: App & PATH Gateway (1 hour)
sidebar_position: 2
---

## App & PATH Gateway Walkthrough via Binary <!-- omit in toc -->

- [What is PATH Gateway?](#what-is-path-gateway)
- [PATH Gateway Operation Requirements](#path-gateway-operation-requirements)
- [Hardware requirements](#hardware-requirements)
- [Kubernetes Example](#kubernetes-example)

## What is PATH Gateway?

See the [PATH Gateway](https://path.grove.city) documentation for more
information on what a `PATH Gateway` is. This page aims to provide links and
details on how to deploy and operate it.

## PATH Gateway Operation Requirements

A PATH Gateway requires the following:

1. A staked onchain [Application](../../3_protocol/actors/2_application.md) to pay for services.
2. An optional onchain [Gateway](../../3_protocol/actors/3_gateway.md) to optionally proxy services.
3. A connection to a [Full Node](2_full_node_docker.md) to interact with the blockchain.

:::tip
It is crucial to deploy a [Full Node](2_full_node_docker.md) prior to setting up a RelayMiner.
This ensures the necessary infrastructure for blockchain communication is in place.
:::

## Hardware requirements

Please see the [Hardware Requirements](../4_faq/6_hardware_requirements.md#path-gateway) page.

## Kubernetes Example

_TODO_DOCUMENT: Provide an example using [strangelove-ventures/cosmos-operator](https://github.com/strangelove-ventures/cosmos-operator)._
