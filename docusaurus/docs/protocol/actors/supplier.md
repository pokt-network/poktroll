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

A `Supplier` is responsible for staking POKT in order to earn POKT in exchange for
providing services as a function of volume and time.

## Schema

The on-chain representation of a `Supplier` can be found at [supplier.proto](https://github.com/pokt-network/poktroll/blob/main/proto/poktroll/shared/supplier.proto).

## Configuration

Configurations to stake an `Supplier` can be found at [supplier_staking_config.md](../../operate/configs/supplier_staking_config.md).

## CLI

The `Supplier` actor depends on both the [`supplier`](https://github.com/pokt-network/poktroll/tree/main/x/supplier)
and [`proof`](https://github.com/pokt-network/poktroll/tree/main/x/proof) on-chain modules.
These two modules' concerns are separated as follows:

### Supplier Module
- Supplier (un/)staking
- Supplier querying

### Proof Module
- Claim creation & querying
- Proof submission & querying

All of the read (i.e. query) based operations for the `Supplier` actor can be
viewed by running the following:

```bash
poktrolld query supplier
```

or

```bash
poktrolld query proof
```

All of the write (i.e. tx) based operations for the `Supplier` actor can be
viewed by running the following:

```bash
poktrolld tx supplier
```

or

```bash
poktrolld tx proof
```
