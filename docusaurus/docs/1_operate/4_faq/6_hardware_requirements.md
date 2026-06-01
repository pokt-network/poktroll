---
title: Hardware Requirements
sidebar_position: 6
---

## Hardware Requirements <!-- omit in toc -->

:::info Last reviewed: 2026-06-01

The numbers below are the canonical hardware reference for this repository's
components. They are reviewed periodically; values for the backend chain node you
serve are owned by that chain's own client documentation (see
[Backend Service Node](#backend-service-node)).

:::

- [Recommended Environment](#recommended-environment)
- [Validator / Full Node](#validator--full-node)
- [RPC Node](#rpc-node)
- [RelayMiner](#relayminer)
- [Backend Service Node (per served chain)](#backend-service-node)
- [PATH Gateway](#path-gateway)
- [Additional Considerations](#additional-considerations)

### Recommended Environment

1. **Linux-based System**: Preferably Debian-based distributions (Ubuntu, Debian).
2. **Architecture Support**: Both x86_64 (amd64) and ARM64 architectures are supported.
3. **Root or Sudo Access**: Administrative privileges are required.
4. **Dedicated Server or Virtual Machine**: Any provider is acceptable.

:::tip Vultr Playbook

If you are using [Vultr](https://www.vultr.com/) for your deployment, you can following the [CLI Playbook we put together here](../5_playbooks/1_vultr.md) to speed things up.

:::

### Validator / Full Node

| Component    | Minimum | Recommended |
| ------------ | ------- | ----------- |
| (v)CPU Cores | 4       | 6           |
| RAM          | 16GB    | 32GB        |
| SSD Storage  | 200GB   | 420GB       |

:::warning In flux

TODO(@okdas): Update these based on the network (Alpha, Beta, MainNet) when taking the latest snapshot.

:::

### RPC Node

If the Full Node will serve as the RPC endpoint for Gateways and RelayMiners under high load, consider:

- Providing more resources
- Deploying multiple Full Nodes for continuous service

### RelayMiner

See the [RelayMiner](../../3_protocol/actors/5_relay_miner.md) documentation for more
information on what a RelayMiner is.

| Component   | Minimum | Recommended |
| ----------- | ------- | ----------- |
| CPU Cores   | 1       | 4           |
| RAM         | 1GB     | 16GB        |
| SSD Storage | 5GB     | 5GB         |

Note that resource requirements for RelayMiner scale linearly with load:

- More suppliers --> Higher resource consumption
- More relays --> Higher resource consumption

:::note

TODO_POST_MAINNET(@okdas): Provide benchmarks for relayminers handling different traffic amounts.

:::

### Backend Service Node (per served chain) {#backend-service-node}

:::critical The biggest, most-often-missed requirement

A `Supplier` earns by proxying relays to a **backend node of the chain it serves**
(e.g. an Ethereum, Base, or Solana node). The `RelayMiner` itself is lightweight
(see the table above), but the backend node it forwards to is **separate
infrastructure that you must run, sync, and maintain** — and it is almost always the
largest cost of operating a Supplier.

The RelayMiner's `5GB` storage figure above does **NOT** include this backend node.

:::

Backend node sizing depends entirely on the chain you serve and grows over time, so
there is no single number. Treat the chain's own client documentation as the source
of truth, and budget for growth. Rough orders of magnitude (verify before deploying):

| Backend chain type           | Typical disk                              | Notes                              |
| ---------------------------- | ----------------------------------------- | ---------------------------------- |
| EVM L1 (e.g. Ethereum)       | hundreds of GB pruned, ~1.2TB+ archive    | Grows continuously                 |
| EVM L2 (e.g. Base, Arbitrum) | ~600GB–2TB+ and growing                   | Fast-growing; size for headroom    |
| Cosmos SDK chains            | tens to hundreds of GB                    | Varies widely by chain and pruning |

Practical guidance:

- Size the disk for **target + 6–12 months of growth**, and prefer providers that
  let you attach or expand block storage after provisioning.
- Sync from a **snapshot** where the chain provides one — a from-genesis sync can
  take days.
- Run the backend node on its own volume (or host) so it does not compete with the
  RelayMiner's claim/proof (`SMT`) storage.

### PATH Gateway

See the [PATH Gateway](https://github.com/pokt-network/path) documentation for more
information on what a `PATH Gateway` is.

| Component   | Minimum | Recommended |
| ----------- | ------- | ----------- |
| CPU Cores   | 1       | 4           |
| RAM         | 1GB     | 16GB        |
| SSD Storage | 5GB     | 5GB         |

<!-- TODO_TECHDEBT: Update the PATH Gateway hardware requirements -->

### Additional Considerations

1. **Scalability**: As your infrastructure grows, you may need to adjust resources accordingly.
2. **Monitoring**: Implement a robust monitoring system to track resource usage and performance.
3. **Redundancy**: For critical operations, consider setting up redundant systems to ensure high availability.
