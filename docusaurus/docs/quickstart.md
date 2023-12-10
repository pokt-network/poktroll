---
sidebar_position: 2
title: Quickstart
---

# Quickstart <!-- omit in toc -->

- [Install Dependencies](#install-dependencies)
- [Develop](#develop)
- [Tools](#tools)
  - [poktrolld](#poktrolld)
  - [Makefile](#makefile)
  - [Ignite](#ignite)
  - [LocalNet](#localnet)

The goal of this document is to get you up and running with the project, a
LocalNet and send an end-to-end relay.

## Install Dependencies

- Install [Docker](https://docs.docker.com/get-docker/)
- Install [Docker Compose](https://docs.docker.com/compose/install/)
- Install [Golang](https://go.dev/doc/install)
- `protoc-gen-go`, `protoc-go-inject-tag` and `mockgen` by running `make install_cli_deps`

1. Mint som new tokens
2. Stake an application
3. Send some funds
4. Send a relay

## Develop

## Tools

### poktrolld

Run `poktrolld --help`

### Makefile

Run `make` to see all the helpers we're working on

### Ignite

### LocalNet

make go_develop_and_test
Please check out the [LocalNet documentation](./localnet/README.md).
