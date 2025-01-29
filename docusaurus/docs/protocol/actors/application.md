---
title: Application Actor
sidebar_position: 2
---

# Application Actor <!-- omit in toc -->

- [Overview](#overview)
- [Schema](#schema)
- [Configuration](#configuration)
- [CLI](#cli)

## Overview

An `Application` is responsible for staking POKT in order to consume and pay for
services available on Pocket Network as a function of volume and time.

## Schema

The onchain representation of an `Application` can be found at [application.proto](https://github.com/pokt-network/poktroll/blob/main/proto/poktroll/application/application.proto).

## Configuration

Configurations to stake an `Application` can be found at [app_staking_config.md](../../operate/configs/app_staking_config.md).

## CLI

All of the read (i.e. query) based operations for the `Application` actor can be
viewed by running:

```bash
poktrolld query application --help
```

All of the write (i.e. tx) based operations for the `Application` actor can be
viewed by running:

```bash
poktrolld tx application --help
```
