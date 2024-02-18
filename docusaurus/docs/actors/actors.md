---
title: Pocket Network Actors
sidebar_position: 1
---

# Pocket Network Actors <!-- omit in toc -->

- [Overview](#overview)
- [On-Chain vs Off-Chain](#on-chain-vs-off-chain)
- [On-Chain Actors](#on-chain-actors)
- [Off-Chain Actors](#off-chain-actors)

## Overview

```mermaid
flowchart TD
    subgraph DA[Data Availability Layer]
        V[Validators]
        B[Blockchain]
    end

    subgraph S[Supplier]
        direction TB
        S1[Supplier 1]
        SN[Supplier N]
    end

    subgraph A[Applications]
        direction TB
        A1[Application 1]
        AN[Application N]
    end

    subgraph G[Gateways]
        direction TB
        P1[Gateway 1]
        PN[Gateway N]
    end

    A <----> |Delegated RPC| G
    G <-- Proxied RPC --> S
    A <--> |Trustless RPC| S

    DA -- Data --> G
    DA -- Data --> A
    DA -- Data --> S

    classDef blue fill:#0000FF
    classDef brown fill:#A52A2A
    classDef red fill:#FF0000
    classDef yellow fill:#DC783D
    classDef acqua fill:#00A3A3
    classDef purple fill:#FF36FF

    class V1,V,VN blue
    class B1,B,BN brown
    class S1,SN yellow
    class P1,PN red
    class F1,FN acqua
    class A1,AN purple
```

## On-Chain vs Off-Chain

Pocket Network protocol is composed of both on-chain and off-chain actors.

There are 3 on-chain actors:

- [Applications](./application.md)
- [Suppliers](./supplier.md)
- [Gateways](./gateway.md)

There are 2 off-chain actors:

- [RelayMiners](./relay_miner.md)
- [AppGateServers](./appgate_server.md)

```mermaid
---
title: Actors
---
flowchart TB

    subgraph on-chain
        A([Application])
        G([Gateway])
        S([Supplier])
    end

    subgraph off-chain
        APS[AppGate Server]
        RM[Relay Miner]
    end

    A -.- APS
    G -.- APS
    S -..- RM
```

## On-Chain Actors

On-Chain actors are part of the Pocket Network distributed ledger. They are the
_"Web3"_ part of Pocket.

They can thought of as a `record`, a `registration` or a piece of `state` at a
certain point in time. They have an `address`, an `account`, a `balance` and often
also have a `stake`.

## Off-Chain Actors

Off-Chain actors are all the operators that make up Pocket Network. They are the
_"Web2"_ part of Pocket.

They can be thought of as `servers`, `processes` or `clients`.

Off-chain actors play a key role in executing off-chain business logic that is
verified on-chain and drives on-chain state transitions.
