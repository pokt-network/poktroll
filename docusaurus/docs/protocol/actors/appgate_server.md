---
title: AppGate Server
sidebar_position: 6
---

# AppGate Server <!-- omit in toc -->

- [Overview](#overview)
- [Configuration](#configuration)
- [CLI](#cli)

## Overview

An `AppGate Server` is responsible for relaying requests between a client dApp
(e.g. mobile app, web app, etc...) and the [Supplier](./supplier.md)s on Pocket
Network, handling all intermediary business logic.

A [Gateway](./gateway.md) operator or a sovereign [Application](./application.md)
interested in accessing Pocket Network directly would need to run an
`AppGate Server` or custom software that implements the same functionality.

## Configuration

Configurations and additional documentation related to operating an `AppGate Server`
can be found at [appgate_server_config.md](../../operate/configs/appgate_server_config.md).

## CLI

All of the operations needed to start and operate an `AppGate Server` can be viewed by running:

```bash
poktrolld appgate-server --help
```
