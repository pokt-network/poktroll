---
title: RelayMiner
sidebar_position: 5
---

# RelayMiner <!-- omit in toc -->

- [Overview](#overview)
- [Configuration](#configuration)
- [CLI](#cli)

## Overview

An `RelayMiner` is responsible for proxying `RelayRequests` between an `AppGate Server`
and the supplied `Service`.

[Suppliers](./supplier.md) interested in providing `Service`s on th Pocket Network
would need to run a `RelayMiner` in addition to the software that provides the said `Service`.

## Configuration

Configurations and additional documentation related to operating a `RelayMiner`
can be found at [relayminer_config.md](../configs/relayminer_config.md).

## CLI

All of the operations needed to start and operate a `RelayMiner` can be viewed
by running:

```bash
poktrolld relayminer --help
```
