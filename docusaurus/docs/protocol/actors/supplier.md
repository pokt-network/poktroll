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

The `Supplier` actor depends on both the `supplier` and `proof` on-chain modules.
The responsibilities of the two modules are as follows:

### Supplier Module
- Supplier staking
- Supplier unstaking
- Supplier querying

### Proof Module
- Claim creation
- Claim querying
- Proof submission
- Proof querying

All of the read (i.e. query) based operations for the `Supplier` actor can be
viewed by running the following, where `<module>` is either "`supplier`" or "`proof`":

```bash
poktrolld query <module>
```

All of the write (i.e. tx) based operations for the `Suplier` actor can be
viewed by running the following, where `<module>` is either "`supplier`" or "`proof`":

```bash
poktrolld tx <module>
```
