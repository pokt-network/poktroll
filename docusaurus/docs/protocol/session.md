---
title: Session
sidebar_position: 1
---

:::warning

TODO(@Olshansk): This is just a placeholder

:::

- [Pre-Requisites](#pre-requisites)
- [Session Duration](#session-duration)
- [Free Work](#free-work)

## Pre-Requisites

There are a number of pre-requisites for the session to be created and for the flow
to function.

1. `Application` must be staked for a specific `Service`
2. `Supplier` must be staked for a specific `Service`
3. `Session` must match `Application` to `Supplier` for the duration of this
   session using on-chain entropy as a pseudorandom seed.

## Session Duration

```mermaid
sequenceDiagram
    actor A as Application(s)
    actor S as Supplier(s)
    participant PN as Pocket Network<br>Distributed Ledger


    A ->> A: Prepare & Sign <br>Relay Request
    A ->> +S: (RelayRequest, AppSig)
    S ->> S: Is App in current session?
    alt Yes: App IS in Supplier's session
        S ->> A: Relay Response
    else No: App IS NOT in Supplier's session
        S ->> A: Error
    end
```

## Free Work

It is i