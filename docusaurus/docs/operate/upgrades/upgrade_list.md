---
title: List of Protocol Upgrades
sidebar_position: 1
---

# List of Protocol Upgrades <!-- omit in toc -->

The tables below provide a list of past and upcoming protocol upgrades. For more detailed information about what upgrades are, how they work, and what changes they bring to the protocol, please refer to our [upgrade overview page](./protocol_upgrades.md).

- [Legend](#legend)
- [MainNet](#mainnet)
- [Beta TestNet](#beta-testnet)
- [Alpha TestNet](#alpha-testnet)
  - [Syncing from genesis - manual steps](#syncing-from-genesis---manual-steps)

## Legend

- ✅ - Yes
- ❌ - No
- ❓ - Unknown/To Be Determined
- ⚠️ - Warning/Caution Required

## MainNet

| Version                                                                  | Planned | Breaking | Requires Manual Intervention | Upgrade Height |
| ------------------------------------------------------------------------ | :-----: | :------: | :--------------------------: | -------------- |
| [`v0.1.1`](https://github.com/pokt-network/poktroll/releases/tag/v0.1.1) |   ✅    |    ✅    |              ❌              | 0              |

## Beta TestNet

| Version                                                                          | Planned | Breaking | Requires Manual Intervention | Upgrade Height |
| -------------------------------------------------------------------------------- | :-----: | :------: | :--------------------------: | -------------- |
| [`v0.0.12`](https://github.com/pokt-network/poktroll/releases/tag/v0.0.12)       |   ✅    |    ✅    |              ❌              | TBA            |
| [`v0.0.11-rc`](https://github.com/pokt-network/poktroll/releases/tag/v0.0.11-rc) |   N/A   |   N/A    |      ❌ genesis version      | N/A            |

## Alpha TestNet

:::warning
Some manual steps are currently required to sync to the latest block. Please follow instructions below.
:::

<!-- DEVELOPER: if important information about the release is changing (e.g. upgrade height is changed) - make sure to update the information in GitHub release as well. -->

| Version                                                                      | Planned | Breaking |                                                          Requires Manual Intervention                                                           | Upgrade Height                                                                                                                  |
| ---------------------------------------------------------------------------- | :-----: | :------: | :---------------------------------------------------------------------------------------------------------------------------------------------: | ------------------------------------------------------------------------------------------------------------------------------- |
| [`v0.0.12`](https://github.com/pokt-network/poktroll/releases/tag/v0.0.12)   |   ✅    |    ✅    |                                                                       ❌                                                                        | TBA                                                                                                                             |
| [`v0.0.11`](https://github.com/pokt-network/poktroll/releases/tag/v0.0.11)   |   ✅    |    ✅    |                                                             ❌ (automatic upgrade)                                                              | [156245](https://shannon.alpha.testnet.pokt.network/pocket/tx/EE72B1D0744872CFFF4AC34DA9573B0BC2E32FFF998A8F25BF817FBE44F53543) |
| [`v0.0.10`](https://github.com/pokt-network/poktroll/releases/tag/v0.0.10)   |   ✅    |    ✅    |                                                             ❌ (automatic upgrade)                                                              | [56860](https://shannon.alpha.testnet.pokt.network/pocket/tx/4E201E5C397AB881F417266154C907D38404BE00BE9A443DE28E44A2B09C5CFB)  |
| [`v0.0.9-4`](https://github.com/pokt-network/poktroll/releases/tag/v0.0.9-4) |   ❌    |    ✅    |                   ⚠️ [follow manual upgrade instructions](https://github.com/pokt-network/poktroll/releases/tag/v0.0.9-4) ⚠️                    | `46329`                                                                                                                         |
| [`v0.0.9-3`](https://github.com/pokt-network/poktroll/releases/tag/v0.0.9-3) |   ❌    |    ✅    | ❌ Active Alpha TestNet Participants Only: [follow manual upgrade instructions](https://github.com/pokt-network/poktroll/releases/tag/v0.0.9-3) | `17102`                                                                                                                         |
| [`v0.0.9`](https://github.com/pokt-network/poktroll/releases/tag/v0.0.9)     |   N/A   |   N/A    |                                                               ❌ genesis version                                                                | N/A                                                                                                                             |

### Syncing from genesis - manual steps

<!-- TODO(@okdas): when the next cosmovisor version released with `https://github.com/cosmos/cosmos-sdk/pull/21790` included - provide automated solution (csv file + pre-downloaded binaries) that will add hot-fixes automatically, allowing to sync from block #1 without any intervention -->

When syncing Alpha TestNet from the first block, the node will fail at height `46329`. Some manual steps are required in order for it to continue. Please [follow manual upgrade instructions](https://github.com/pokt-network/poktroll/releases/tag/v0.0.9-4).
