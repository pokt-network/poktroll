---
title: RelayMiner
sidebar_position: 5
---

# RelayMiner <!-- omit in toc -->

- [Overview](#overview)
- [Configuration](#configuration)
- [CLI](#cli)

## Overview

An `RelayMiner` is responsible for  for relaying requests between a client dApp
(e.g. mobile app, web app, etc...) and the `Supplier`s on Pocket Network, handling
all intermediary business logic.

A `Gateway` operator or a sovereign `Application` interested in accessing Pocket
Network directly would need to run an `AppGate Server` or custom software that
implements the same functionality.

## Configuration

Configurations to stake an `AppGate Server` can be found [app_staking_config.md](../configs/appgate_server_config.md).

## CLI

All of the operations needed to start and operate an `AppGate Server` can be viewed by running:

```bash
poktrolld relayminer --help
```
