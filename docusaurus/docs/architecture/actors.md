---
title: Pocket Network Actors
sidebar_position: 2
---

- On-Chain (stake / unstake); part of the protocol
  - Registered
  - Gateway
  - Application
  - Supplier (Servicer State)
    - On-chain state, not a process
    - A registered RelayMiner
- Off-Chain (configurations / operations); part of the SDK
  - Clients
  - RelayMiner (Servicer)
    - (operates alongside a supplier)
    - Process that a supplier runs
    - All suppliers need a RelayMiner to function
  - AppGateServer (operates alongside an Application and/or Gateway)

# Pocket Network Actors <!-- omit in toc -->

- [Overview](#overview)

## Overview

The Pocket Network protocol is composed of a set of on-chain actors:

- **Applications**:
- **Suppliers**:
- **Gateways**:

This poktroll repository provides two

The Pocket Network actors can be split into two types:

Pocket Network enables a Utilitarian economy that proportionally incentivizes or penalizes the corresponding infrastructure providers based on their quality of service. It is composed of the following actors:

Staked Applications that purchase Web3 access over a function of volume and time
Staked Servicers that earn rewards for providing Web3 access over a function of volume and quality
Elected Fishermen who grade and enforce the quality of the Web3 access provided by Servicers
Staked Validators responsible for maintaining safety & liveness of the replicated state machine
Registered Gateways that can be optionally leveraged by Applications through delegated trust
