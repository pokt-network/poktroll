---
title: Gateway Actor
sidebar_position: 3
---

# Gateway Actor <!-- omit in toc -->

- [Overview](#overview)
- [Schema](#schema)
- [Configuration](#configuration)
- [CLI](#cli)

## Overview

A `Gateway` is responsible for staking POKT in order to relay and sign requests
on behalf of an [Application](./application.md).

## Schema

The onchain representation of a `Gateway` can be found at [gateway.proto](https://github.com/pokt-network/poktroll/blob/main/proto/poktroll/gateway/gateway.proto).

## Configuration

Configurations to stake a `Gateway` can be found at [gateway_staking_config.md](../../operate/configs/gateway_staking_config.md).

## CLI

All of the read (i.e. query) based operations for the `Gateway` actor can be
viewed by running:

```bash
poktrolld query gateway --help
```

All of the write (i.e. tx) based operations for the `Gateway` actor can be
viewed by running:

```bash
poktrolld tx gateway --help
```
