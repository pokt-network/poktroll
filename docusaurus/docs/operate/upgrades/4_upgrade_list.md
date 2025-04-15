---
title: Protocol Upgrade List
sidebar_position: 4
---

The tables below provide a list of past and upcoming protocol upgrades.

For more detailed information about what upgrades are, how they work, and what changes they bring to the protocol, please refer to our [upgrade overview page](1_protocol_upgrades.md).

## Table of Contents

- [Table of Contents](#table-of-contents)
- [Legend](#legend)
- [MainNet Protocol Upgrades](#mainnet-protocol-upgrades)
- [Beta TestNet Protocol Upgrades](#beta-testnet-protocol-upgrades)
- [Alpha TestNet Protocol Upgrades](#alpha-testnet-protocol-upgrades)
  - [Syncing from genesis - manual steps](#syncing-from-genesis---manual-steps)

## Legend

- ✅ - Yes
- ❌ - No
- ❓ - Unknown/To Be Determined
- ⚠️ - Warning/Caution Required

## MainNet Protocol Upgrades

| Version                                                                  | Planned | Breaking | Requires Manual Intervention | Upgrade Height |
| ------------------------------------------------------------------------ | :-----: | :------: | :--------------------------: | -------------- |
| [`v0.1.2`](https://github.com/pokt-network/poktroll/releases/tag/v0.1.2) |   ✅    |    ✅    |              ❓              | TBA            |
| [`v0.1.1`](https://github.com/pokt-network/poktroll/releases/tag/v0.1.1) |   ✅    |    ✅    |              ❌              | 0              |

## Beta TestNet Protocol Upgrades

| Version                                                                          | Planned | Breaking | Requires Manual Intervention | Upgrade Height |
| -------------------------------------------------------------------------------- | :-----: | :------: | :--------------------------: | -------------- |
| [`v0.1.2`](https://github.com/pokt-network/poktroll/releases/tag/v0.1.2)         |   ✅    |    ✅    |              ❓              | TBA            |
| [`v0.1.1`](https://github.com/pokt-network/poktroll/releases/tag/v0.1.1)         |   ✅    |    ✅    |              ❌              | TODO(@okdas)   |
| [`v0.0.12`](https://github.com/pokt-network/poktroll/releases/tag/v0.0.12)       |   ✅    |    ✅    |              ❌              | TODO(@okdas)   |
| [`v0.0.11-rc`](https://github.com/pokt-network/poktroll/releases/tag/v0.0.11-rc) |   N/A   |   N/A    |      ❌ genesis version      | 0              |

## Alpha TestNet Protocol Upgrades

:::warning TODO_TECHDEBT(@okdas)

Review this section and the one below it to remove/update accordingly.

:::

| Version                                                                      | Planned | Breaking |                                                          Requires Manual Intervention                                                           | Upgrade Height                                                                                                                  |
| ---------------------------------------------------------------------------- | :-----: | :------: | :---------------------------------------------------------------------------------------------------------------------------------------------: | ------------------------------------------------------------------------------------------------------------------------------- |
| [`v0.1.2`](https://github.com/pokt-network/poktroll/releases/tag/v0.1.2)     |   ✅    |    ✅    |                                                                       ❓                                                                        | TBA                                                                                                                             |
| [`v0.1.1`](https://github.com/pokt-network/poktroll/releases/tag/v0.1.1)     |   ✅    |    ✅    |                                                                       ❌                                                                        | TODO(@okdas)                                                                                                                    |
| [`v0.0.12`](https://github.com/pokt-network/poktroll/releases/tag/v0.0.12)   |   ✅    |    ✅    |                                                                       ❌                                                                        | TODO(@okdas)                                                                                                                    |
| [`v0.0.11`](https://github.com/pokt-network/poktroll/releases/tag/v0.0.11)   |   ✅    |    ✅    |                                                             ❌ (automatic upgrade)                                                              | [156245](https://shannon.alpha.testnet.pokt.network/pocket/tx/EE72B1D0744872CFFF4AC34DA9573B0BC2E32FFF998A8F25BF817FBE44F53543) |
| [`v0.0.10`](https://github.com/pokt-network/poktroll/releases/tag/v0.0.10)   |   ✅    |    ✅    |                                                             ❌ (automatic upgrade)                                                              | [56860](https://shannon.alpha.testnet.pokt.network/pocket/tx/4E201E5C397AB881F417266154C907D38404BE00BE9A443DE28E44A2B09C5CFB)  |
| [`v0.0.9-4`](https://github.com/pokt-network/poktroll/releases/tag/v0.0.9-4) |   ❌    |    ✅    |                   ⚠️ [follow manual upgrade instructions](https://github.com/pokt-network/poktroll/releases/tag/v0.0.9-4) ⚠️                    | `46329`                                                                                                                         |
| [`v0.0.9-3`](https://github.com/pokt-network/poktroll/releases/tag/v0.0.9-3) |   ❌    |    ✅    | ❌ Active Alpha TestNet Participants Only: [follow manual upgrade instructions](https://github.com/pokt-network/poktroll/releases/tag/v0.0.9-3) | `17102`                                                                                                                         |
| [`v0.0.9`](https://github.com/pokt-network/poktroll/releases/tag/v0.0.9)     |   N/A   |   N/A    |                                                               ❌ genesis version                                                                | N/A                                                                                                                             |

### Syncing from genesis - manual steps

<!-- TODO(@okdas): when the next cosmovisor version released with `https://github.com/cosmos/cosmos-sdk/pull/21790` included - provide automated solution (csv file + pre-downloaded binaries) that will add hot-fixes automatically, allowing to sync from block #1 without any intervention -->

When syncing Alpha TestNet from the first block, the node will fail at height `46329`. Some manual steps are required in order for it to continue. Please [follow manual upgrade instructions](https://github.com/pokt-network/poktroll/releases/tag/v0.0.9-4).
