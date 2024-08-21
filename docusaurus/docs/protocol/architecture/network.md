---
title: Outdated - Pocket Actors, Nodes & Data Availability Network
sidebar_position: 1
---

:::danger
TODO(@Olshansk): This file was copied over from the `poktroll-alpha` repo and has
not been updated to reflect recent changse & learnings.
:::

# Pocket Nodes & Validators <!-- omit in toc -->

- [Dependant Node](#dependant-node)
- [Sovereign Node](#sovereign-node)

This document aims to show a high level diagram of the nodes participating in the Pocket Network.

It includes the flow of Requests, Data, Transactions, and Blocks.

## Dependant Node

The diagram below shows the absolute base case where there is:

1. Pocket Full Node
2. The Full Node is also the Single Validator in the network
3. The Single Validator is also the Proxy's (i.e. Relayer/Miner) source of data and events

A Dependant Relayer is one that:

- Sends Txs to the validator (or another node that gossips with the validator)
- Trusts another node to:
  - read on-chain data
  - listen for on-chain events

```mermaid
---
title: Dependant Relayer
---
flowchart TB
    a(("Application"))
    subgraph p["Pocket Node"]
        direction LR
        rs([Role 1 - Validator])
        rv([Role 2 - Servicer])
        pl1[("Pocket Full Node")]
    end
    subgraph r["Relayer (off-chain)"]
        direction TB
        eth[["Ethereum"]]
        gn[["Gnosis"]]
        pg[["Polygon"]]
        etc[["..."]]
    end
    da{"Pocket Network DA"}
    a -- RPC Relay Req/Res \n (JSON-RPC endpoint) --> r
    p -. Block & Tx Events \n (Websocket listener).-> r
    r -- Session Dispatch Req/Res \n (JSON-RPC endpoint)--> p
    r -. Txs \n (JSON-RPC endpoint).-> p
    p -. Blocks (Commit) .-> da
    da -. Blocks (Sync) .-> p
```

## Sovereign Node

The diagram below shows the Pocket Network DA, Validators, Full Nodes and Actors.

A Sovereign Relayer is one that:

- Sends Txs to the validator (or a node that gossips with the validators)
- Runs it's own Pocket Full Node to:
  - read on-chain data
  - listen for on-chain events

```mermaid
---
title: Sovereign Servicer
---
flowchart TB
    a(("Application"))
    subgraph pfn["Pocket Full Nodes"]
        pfn1[("Pocket Full Node")]
        pfn2[("Pocket Full Node")]
        pfn3[("Pocket Full Node")]
        pfn1 <-. gossip \n (Txs & Blocks) .-> pfn2
        pfn2 <-. gossip \n (Txs & Blocks) .-> pfn3
        pfn3 <-. gossip \n (Txs & Blocks) .-> pfn1
    end
    subgraph pv["Validator"]
        pl1[("Pocket Full Node")]
    end
    subgraph r["Proxy (off-chain Relayer & Miner)"]
        direction TB
        eth[["Ethereum"]]
        gn[["Gnosis"]]
        pg[["Polygon"]]
        etc[["..."]]
    end
    subgraph s["Servicer (Full Node maintained by Proxy Operator) "]
        pl2[("Pocket Full Node")]
    end
    da{"Data Availability"}
    a -- Relay Req/Res \n (JSON-RPC endpoint) --> r
    s -. Block & Tx Events \n (Websocket listener).-> r
    r -- Session Dispatch Req/Res \n (JSON-RPC endpoint)--> s
    r -. Txs \n (JSON-RPC endpoint).-> pfn
    r -. Txs \n (JSON-RPC endpoint).-> pv
    pfn <-. gossip \n (Txs & Blocks) .-> pv
    pv -. Blocks\n(Commit) .->da
    da -. Blocks\n(Sync) .-> pv
    da -. Blocks\n(Sync).-> s
```
