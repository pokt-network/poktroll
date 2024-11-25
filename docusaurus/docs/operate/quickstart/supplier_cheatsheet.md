---
sidebar_position: 4
title: Supplier (RelayMiner) Cheat Sheet
---

## Supplier Cheat Sheet <!-- omit in toc -->

- [Context](#context)
- [Pre-Requisites](#pre-requisites)
- [1. Deploy your Server \& Install Dependencies](#1-deploy-your-server--install-dependencies)
- [1. Retrieve the source code](#1-retrieve-the-source-code)

### Context

This document is a cheat sheet to get you quickly started with two things:

1. Staking an onchain `Supplier`
2. Deploying an offchain `RelayMiner`

By the end of it, you should be able to serve Relays onchain.

:::tip

It is intended to be a < 10 minute quick copy-pasta.

If you're interested in spending hours reading and understanding how things work,
please see the [Supplier Walkthrough](./../run_a_node/supplier_walkthrough.md)

:::

### Pre-Requisites

You will need the following:

1. A funded onchain wallet/account/address
2. A known service

- How do I stake a service?
- How do I view existing services?
- How do I

### 1. Deploy your Server & Install Dependencies

You can deploy a RelayMiner on any server.

If you are just getting started, you can follow along the team at Grove and follow
the instructions in the [Docker Compose Cheat Sheet](./docker_compose_debian_cheatsheet#deploy-your-server)
to deploy a Debian server on a Vultr instance.

### 1. Retrieve the source code

```bash
mkdir ~/workspace && cd ~/workspace
git clone https://github.com/pokt-network/poktroll-docker-compose-example.git
cd poktroll-docker-compose-example
```
