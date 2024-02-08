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

An `Gateway` is responsible for staking POKT in order to relay and sign
requests on behalf of an `Application`.

The `Application` is responsible for staking POKT to consume and pay for services
available on Pocket Network as a function of volume and time, but the `Gateway`
can help facilitate that access.

## Schema

The on-chain for an `Gateway` can be found at [gateway.proto](./../../../proto/pocket/gateway/gateway.proto).

## Configuration

Configurations to stake an `Gateway` can be found [gateway_staking_config.md](../configs/gateway_staking_config.md).

## CLI

All of the read (i.e. query) based operations for the `Gateway` actor can be
viewed by running:

```bash
poktrolld query gateway --help
```

All of the write (i.e. tx) based operations for the `Application` actor can be
viewed by running:

```bash
poktrolld tx gateway --help
```
