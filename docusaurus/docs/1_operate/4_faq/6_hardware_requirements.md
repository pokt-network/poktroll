---
title: Hardware Requirements
sidebar_position: 6
---

## Hardware Requirements <!-- omit in toc -->

:::warning
We are continuously evaluating the hardware requirements as we work on the next version of Pocket Network.

TODO_MAINNET: Update this document prior to MainNet release
:::

- [Recommended Environment](#recommended-environment)
- [Validator / Full Node](#validator--full-node)
- [RPC Node](#rpc-node)
- [RelayMiner](#relayminer)
- [PATH Gateway](#path-gateway)
- [Additional Considerations](#additional-considerations)

### Recommended Environment

1. **Linux-based System**: Preferably Debian-based distributions (Ubuntu, Debian).
2. **Architecture Support**: Both x86_64 (amd64) and ARM64 architectures are supported.
3. **Root or Sudo Access**: Administrative privileges are required.
4. **Dedicated Server or Virtual Machine**: Any provider is acceptable.

:::tip Vultr Playbook

If you are using [Vultr](https://www.vultr.com/) for your deployment, you can following the [CLI Playbook we put together here](../walkthroughs/playbooks/vultr.md) to speed things up.

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

See the [RelayMiner](../../4_protocol/actors/5_relay_miner.md) documentation for more
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

### PATH Gateway

See the [PATH Gateway](https://path.grove.city) documentation for more
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
