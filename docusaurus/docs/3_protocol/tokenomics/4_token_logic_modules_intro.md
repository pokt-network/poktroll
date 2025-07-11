---
title: Token Logic Modules Introduction
sidebar_position: 4
---

- [Introduction](#introduction)
- [Background: Max Claimable Amount](#background-max-claimable-amount)
- [TLM (pre) Processing](#tlm-pre-processing)

## Introduction

Token Logic Module (TLM) processing consists of the following sequential steps:

1. `TLM pre-processing` - General pre-processing to determine the number of tokens to settle per claim.
2. `TLM processing` - Iterating through each TLM, sequentially, independently of each other.

## Background: Max Claimable Amount

_tl;dr Max Claimable Amount ∝ (Application Stake / Number of Suppliers per Session)_

Per **Algorithm 1** of the [Relay Mining paper](https://arxiv.org/pdf/2305.10672),
the maximum amount a supplier can claim from an application in a single session
MUST NOT exceed the Application's stake divided by the number of suppliers in the session.

This is referred to as "Relay Mining Payable Relay Accumulation" in the paper and
is described by the following pseudo-code:

![Algorithm 1](https://github.com/user-attachments/assets/d1a61535-aa31-447d-88ea-c8d14dcb20c6)

:::tip

See the [relay mining docs](../primitives/relay_mining.md) or the [annotated presentation](https://olshansky.substack.com/p/annotated-presentation-relay-mining) for more information.

:::

## TLM (pre) Processing

_tl;dr Determine if the claim amount is greater than the maximum claimable amount prior to running each individual TLM._

**Prior to** processing each individual TLM, we need to understand if the amount claimed
by the supplier adheres to the optimistic maxIA set per the limits of the Relay Mining algorithm.

:::info

Pocket Network can be seen as a probabilistic, optimistic permissionless multi-tenant rate limiter.

This works by putting funds in escrow, burning it after work is done, and putting optimistic limits
in place whose work volume is proven onchain.

:::

Suppliers always have the option to over-service an Application (**i.e. do free work**),
in order to ensure high quality service in the network. This may lead to offchain
reputation benefits (e.g. Gateways favoring them), but suppliers' onchain rewards
are always limited by the cumulative amounts Applications' stakes (at session start; per service)
and the number of Suppliers in the session.

```mermaid
---
title: "Token Logic Modules Pre-Processing"
---
flowchart TB
    CA(["Claim Amount (CA)"])
    MCA(["Mac Claimable Amount (MCA) <br> = (AppStake / NumSuppliersPerSession)"])
    CC{"Is CA > MCA?"}
    Update(Broadcast Event <br> that SA = MCA)
    SOAE{{Application Overserviced <br> Event}}
    TLMP("Process TLMs <br> Settlement Amount (SA)")

    CA -- CA --> CC
    MCA -- MCA --> CC

    Update -..-> SOAE
    CC -- Yes --> Update
    CC -- No<br>SA=CA --> TLMP
    Update -- SA=MCA --> TLMP

    TLMP --SA--> TLMBEM[[TLM: Burn Equals Mint]]
    TLMP --SA--> TLMGI[[TLM: Global Inflation]]
    TLMP --SA--> TLMO[[TLM: ...]]
    TLMGI --> TLMGIRR[[TLM: Global Inflation Reimbursement Request]]

    classDef tlm fill:#54ebd5,color: #333,stroke:#333,stroke-width:2px;
    classDef question fill:#e3db6d,color: #333,stroke:#333,stroke-width:2px;
    classDef event fill:#e8b761,color: #333,stroke:#333,stroke-width:2px;

    class TLMBEM,TLMGI,TLMGIRR,TLMO tlm;
    class SOAE event;
    class CC question;
```

:::warning

In order for the `MaxClaimableAmount` to prevent Applications from over-servicing,
the `Application.Stake` must be claimable only by `Supplier`s from the same session
(i.e. the same service).

For a given application `MaxClaimableAmount` is `Application.Stake / NumSuppliersPerSession`
and defined per session/service.

If an `Application` is able send traffic to `n` services then it could be over-servicing
up to `n` times its stake for a given session number by performing
`MaxClaimableAmount * NumSuppliersPerSession * n > Application.Stake` worth of work.

To avoid thy type of over-servicing, The Pocket protocol requires `Application`s
to only be able to stake for EXACTLY ONE service.

:::
