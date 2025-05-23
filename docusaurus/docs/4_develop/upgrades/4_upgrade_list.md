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

| Version                                                                    | Planned | Breaking | Requires Manual Intervention | Upgrade Height  |
|----------------------------------------------------------------------------| :-----: |:--------:|:----------------------------:|:----------------|
| [`v0.1.12`](https://github.com/pokt-network/poktroll/releases/tag/v0.1.12) |   ✅    |    ❌    |              ❓              | TBA             |
| [`v0.1.11`](https://github.com/pokt-network/poktroll/releases/tag/v0.1.11) |   ✅    |    ✅    |              ❓              | TBA             |
| [`v0.1.7`](https://github.com/pokt-network/poktroll/releases/tag/v0.1.7)   |   ✅    |    ❌    |              ❓              | TBA             |
| [`v0.1.6`](https://github.com/pokt-network/poktroll/releases/tag/v0.1.6)   |   ✅    |    ✅    |              ❓              | TBA             |
| [`v0.1.5`](https://github.com/pokt-network/poktroll/releases/tag/v0.1.5)   |   ✅    |    ✅    |              ❓              | TBA             |
| [`v0.1.4`](https://github.com/pokt-network/poktroll/releases/tag/v0.1.4)   |   ✅    |    ✅    |              ❓              | TBA             |
| [`v0.1.3`](https://github.com/pokt-network/poktroll/releases/tag/v0.1.3)   |   ✅    |    ✅    |              ❓              | TBA             |
| [`v0.1.2`](https://github.com/pokt-network/poktroll/releases/tag/v0.1.2)   |   ✅    |    ✅    |              ❓              | TBA             |
| [`v0.1.1`](https://github.com/pokt-network/poktroll/releases/tag/v0.1.1)   |   ✅    |    ✅    |     ❌ (genesis version)     | `0`             |

## Beta TestNet Protocol Upgrades

| Version                                                                          | Planned | Breaking | Requires Manual Intervention | Upgrade Height                                                                                                                    |
|----------------------------------------------------------------------------------|:-------:|:--------:|:----------------------------:|:----------------------------------------------------------------------------------------------------------------------------------|
| [`v0.1.12`](https://github.com/pokt-network/poktroll/releases/tag/v0.1.12)       |    ❌    |    ❌     |             ❌                | [`14812`](https://shannon.beta.testnet.pokt.network/poktroll/tx/87E3C205C5991C39468FDFA969C85A98A8770754623B638033622E749378D814) |
| [`v0.1.12`](https://github.com/pokt-network/poktroll/releases/tag/v0.1.12)       |    ✅    |    ❌     | Added to the [skip upgrade heights list](https://github.com/pokt-network/pocket-network-genesis/blob/master/shannon/testnet-beta/skip_upgrade_heights) due to a bad release archive file                           | [`14790`](https://shannon.beta.testnet.pokt.network/poktroll/tx/5A32931F4F287B9100C928F54ABEA98F896B68038335B6860E5F784423060A04) |
| [`v0.1.11`](https://github.com/pokt-network/poktroll/releases/tag/v0.1.11)       |    ✅    |    ✅     |              ❌               | [`11100`](https://shannon.beta.testnet.pokt.network/poktroll/tx/652AA6EA6DC99FA2448B8402DE376F24058C6F48956FBBFFA67D06388899EE5E) |
| [`v0.1.7`](https://github.com/pokt-network/poktroll/releases/tag/v0.1.7)         |    ✅    |    ❌     |              ❌               | `6388`                                                                                                                            |
| [`v0.1.6`](https://github.com/pokt-network/poktroll/releases/tag/v0.1.6)         |    ✅    |    ✅     |              ❌               | `6110`                                                                                                                            |
| [`v0.1.5`](https://github.com/pokt-network/poktroll/releases/tag/v0.1.5)         |    ✅    |    ✅     |              ❌               | `5831`                                                                                                                            |
| [`v0.1.4`](https://github.com/pokt-network/poktroll/releases/tag/v0.1.4)         |    ✅    |    ✅     |              ❌               | `4596`                                                                                                                            |
| [`v0.1.3`](https://github.com/pokt-network/poktroll/releases/tag/v0.1.3)         |    ✅    |    ✅     |              ❌               | `4022`                                                                                                                            |
| [`v0.1.2`](https://github.com/pokt-network/poktroll/releases/tag/v0.1.2)         |    ✅    |    ✅     |              ❓               | TBA                                                                                                                               |
| [`v0.1.1`](https://github.com/pokt-network/poktroll/releases/tag/v0.1.1)         |    ✅    |    ✅     |              ❌               | TODO(@okdas)                                                                                                                      |
| [`v0.0.12`](https://github.com/pokt-network/poktroll/releases/tag/v0.0.12)       |    ✅    |    ✅     |              ❌               | TODO(@okdas)                                                                                                                      |
| [`v0.0.11-rc`](https://github.com/pokt-network/poktroll/releases/tag/v0.0.11-rc) |   N/A   |   N/A    |     ❌ (genesis version)      | `0`                                                                                                                               |

## Alpha TestNet Protocol Upgrades

:::warning TODO_TECHDEBT(@okdas)

Review this section and the one below it to remove/update accordingly.

:::

| Version                                                                      | Planned | Breaking | Requires Manual Intervention | Upgrade Height                                                                                                                     | Notes                                                                                                                                        |
|------------------------------------------------------------------------------|:-------:|:--------:|:----------------------------:|:-----------------------------------------------------------------------------------------------------------------------------------|:---------------------------------------------------------------------------------------------------------------------------------------------|
| [`v0.1.12`](https://github.com/pokt-network/poktroll/releases/tag/v0.1.12)   |    ✅    |    ✅     |              ❌               | [`73055`](https://shannon.alpha.testnet.pokt.network/poktroll/tx/F9643B2F7F769CC6DA7F8761B607E3D059F68CC4425AB0DCF2EB0E0E89D08E05) |                                                                                                                                              |
| [`v0.1.11`](https://github.com/pokt-network/poktroll/releases/tag/v0.1.11)   |    ✅    |    ✅     |              ❌               | [`55246`](https://shannon.alpha.testnet.pokt.network/poktroll/tx/72CD719FDBFA29E03CE4139CA3BFF87D847099B92BBBE4CEC14C96ADE7DB2509) |                                                                                                                                              |
| [`v0.1.7`](https://github.com/pokt-network/poktroll/releases/tag/v0.1.7)     |    ✅    |    ❌     |              ❌               | `33308`                                                                                                                            |                                                                                                                                              |
| [`v0.1.6`](https://github.com/pokt-network/poktroll/releases/tag/v0.1.6)     |    ✅    |    ✅     |              ❌               | `32979`                                                                                                                            |                                                                                                                                              |
| [`v0.1.5`](https://github.com/pokt-network/poktroll/releases/tag/v0.1.5)     |    ✅    |    ✅     |              ❌               | `31597`                                                                                                                            |                                                                                                                                              |
| [`v0.1.4`](https://github.com/pokt-network/poktroll/releases/tag/v0.1.4)     |    ✅    |    ✅     |              ❌               | `25499`                                                                                                                            |                                                                                                                                              |
| [`v0.1.3`](https://github.com/pokt-network/poktroll/releases/tag/v0.1.3)     |    ✅    |    ✅     |              ❌               | `22634`                                                                                                                            |                                                                                                                                              |
| [`v0.1.2`](https://github.com/pokt-network/poktroll/releases/tag/v0.1.2)     |    ✅    |    ✅     |              ❓               | `21515`                                                                                                                            |                                                                                                                                              |
| [`v0.1.1`](https://github.com/pokt-network/poktroll/releases/tag/v0.1.1)     |    ✅    |    ✅     |              ❌               | TODO(@okdas)                                                                                                                       |                                                                                                                                              |
| [`v0.0.12`](https://github.com/pokt-network/poktroll/releases/tag/v0.0.12)   |    ✅    |    ✅     |              ❌               | TODO(@okdas)                                                                                                                       |                                                                                                                                              |
| [`v0.0.11`](https://github.com/pokt-network/poktroll/releases/tag/v0.0.11)   |    ✅    |    ✅     |              ❌               | [`156245`](https://shannon.alpha.testnet.pokt.network/pocket/tx/EE72B1D0744872CFFF4AC34DA9573B0BC2E32FFF998A8F25BF817FBE44F53543)  |
| [`v0.0.10`](https://github.com/pokt-network/poktroll/releases/tag/v0.0.10)   |    ✅    |    ✅     |              ❌               | [`56860`](https://shannon.alpha.testnet.pokt.network/pocket/tx/4E201E5C397AB881F417266154C907D38404BE00BE9A443DE28E44A2B09C5CFB)   |
| [`v0.0.9-4`](https://github.com/pokt-network/poktroll/releases/tag/v0.0.9-4) |    ❌    |    ✅     |              ⚠️              | `46329`                                                                                                                            | ⚠️ [follow manual upgrade instructions](https://github.com/pokt-network/poktroll/releases/tag/v0.0.9-4) ⚠️                                   |
| [`v0.0.9-3`](https://github.com/pokt-network/poktroll/releases/tag/v0.0.9-3) |    ❌    |    ✅     |              ❌               | `17102`                                                                                                                            | Active Alpha TestNet Participants Only: [follow manual upgrade instructions](https://github.com/pokt-network/poktroll/releases/tag/v0.0.9-3) |
| [`v0.0.9`](https://github.com/pokt-network/poktroll/releases/tag/v0.0.9)     |   N/A   |   N/A    |              ❌               | N/A                                                                                                                                | genesis version                                                                                                                              |

### Syncing from genesis - manual steps

<!-- TODO(@okdas): when the next cosmovisor version released with `https://github.com/cosmos/cosmos-sdk/pull/21790` included - provide automated solution (csv file + pre-downloaded binaries) that will add hot-fixes automatically, allowing to sync from block #1 without any intervention -->

When syncing Alpha TestNet from the first block, the node will fail at height `46329`. Some manual steps are required in order for it to continue. Please [follow manual upgrade instructions](https://github.com/pokt-network/poktroll/releases/tag/v0.0.9-4).
