---
title: Hardware Requirements
sidebar_position: 1
---

## Hardware Requirements <!-- omit in toc -->

:::warning
We are continuously evaluating the hardware requirements as we work on the next version of Pocket Network.

TODO_MAINNET: Update this document prior to MainNet release
:::

- [Validator / Full Node](#validator--full-node)
- [RPC Node](#rpc-node)
- [RelayMiner](#relayminer)
- [AppGate Server / Gateway](#appgate-server--gateway)
- [Additional Considerations](#additional-considerations)

### Validator / Full Node

| Component   | Minimum | Recommended |
| ----------- | ------- | ----------- |
| CPU Cores   | 2       | 4           |
| RAM         | 8GB     | 16GB        |
| SSD Storage | 50GB    | 50GB        |

### RPC Node

If the Full Node will serve as the RPC endpoint for Gateways and RelayMiners under high load, consider:

- Providing more resources
- Deploying multiple Full Nodes for continuous service

### RelayMiner

See the [RelayMiner](../../protocol/actors/appgate_server.md) documentation for more
information on what a RelayMiner is.

| Component   | Minimum | Recommended |
| ----------- | ------- | ----------- |
| CPU Cores   | 1       | 4           |
| RAM         | 1GB     | 16GB        |
| SSD Storage | 5GB     | 5GB         |

Note that resource requirements for RelayMiner scale linearly with load:

- More suppliers --> Higher resource consumption
- More relays --> HIgher resource consumption

:::note

TODO_POST_MAINNET(@okdas): Provide benchmarks for relayminers handling different traffic amounts.

:::

### AppGate Server / Gateway

See the [AppGate Server](../../protocol/actors/appgate_server.md) documentation for more
information on what an AppGate Server is.

| Component   | Minimum | Recommended |
| ----------- | ------- | ----------- |
| CPU Cores   | 1       | 4           |
| RAM         | 1GB     | 16GB        |
| SSD Storage | N/A     | N/A         |

**Note**: This service is stateless and does not require SSD storage.

### Additional Considerations

1. **Scalability**: As your infrastructure grows, you may need to adjust resources accordingly.
2. **Monitoring**: Implement a robust monitoring system to track resource usage and performance.
3. **Redundancy**: For critical operations, consider setting up redundant systems to ensure high availability.
