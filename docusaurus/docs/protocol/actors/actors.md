---
title: Pocket Network Actors
sidebar_position: 1
---

# Pocket Network Actors <!-- omit in toc -->

- [Overview](#overview)
- [On-Chain Actors](#on-chain-actors)
  - [Risks \& Misbehavior](#risks--misbehavior)
- [Off-Chain Actors](#off-chain-actors)

## Overview

Pocket Network protocol is composed of both on-chain and off-chain actors.

There are 3 on-chain actors:

- [Applications](./application.md)
- [Suppliers](./supplier.md)
- [Gateways](./gateway.md)

There are 2 off-chain actors:

- [RelayMiners](./relay_miner.md)
- [PATH Gateways](./path_gateway.md)

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
        PG[PATH Gateway]
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

### Risks & Misbehavior

:::warning
This is an open work in progress and an active area of research.
:::

```mermaid
mindmap
    (On-Chain Actors)
        Gateway
            Risks
                Intentional overservicing
                Off-chain only?
            Misbehavior
                Low volume exploit
                On-chain, there are few/any? expectations of gateway actors; basically a registry to track gateways and application delegations
                On-chain, we cannot robustly distinguish requests sent by gateways from those sent by applications acting sovereignly
        Application
            Risks
                Insufficient funds to pay for services received
                Intentional overservicing
            Misbehavior
                Low volume exploit
        Supplier
            Risks
                Service/quality degredation
            Misbehavior
                No or low quality responses to valid requests for service
                Invalid/missing proofs
```

## Off-Chain Actors

Off-Chain actors are all the operators that make up Pocket Network. They are the
_"Web2"_ part of Pocket.

They can be thought of as `servers`, `processes` or `clients`.

Off-chain actors play a key role in executing off-chain business logic that is
verified on-chain and drives on-chain state transitions.
