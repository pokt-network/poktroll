---
title: Supplier Actor
sidebar_position: 4
---

# Supplier Actor <!-- omit in toc -->

- [Overview](#overview)
- [Schema](#schema)
- [Configuration](#configuration)
- [CLI](#cli)

## Overview

An `Supplier` is responsible for staking POKT in order to consume and pay for
services available on Pocket Network as a function of volume and time.

## Schema

The on-chain for an `Supplier` can be found at [supplier.proto](./../../../proto/pocket/supplier/supplier.proto).

## Configuration

Configurations to stake an `Supplier` can be found [app_staking_config.md](../configs/supplier_staking_config.md).

## CLI

All of the read (i.e. query) based operations for the `Supplier` actor can be
viewed by running:

```bash
poktrolld query supplier
```

All of the write (i.e. tx) based operations for the `Application` actor can be
viewed by running:

```bash
poktrolld tx supplier
```
