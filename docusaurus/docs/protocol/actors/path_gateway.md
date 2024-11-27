---
title: PATH Gateway
sidebar_position: 6
---

# PATH Gateway <!-- omit in toc -->

- [Overview](#overview)
- [Configuration](#configuration)
- [CLI](#cli)

## Overview

`PATH Gateway` is responsible for relaying requests between a client dApp
(e.g. mobile app, web app, etc...) and the [Supplier](./supplier.md)s on Pocket
Network, handling all intermediary business logic.

A [Gateway](./gateway.md) operator or a sovereign [Application](./application.md)
interested in accessing Pocket Network directly would need to run a `PATH Gateway`
or custom software that implements the same functionality.

## Configuration

Configurations and additional documentation related to operating a `PATH Gateway`
can be found at [path_gateway.md](https://path.grove.city/operate).

## CLI

All of the operations needed to start and operate a `PATH Gateway` can be viewed by running:

```bash
path --help
```
