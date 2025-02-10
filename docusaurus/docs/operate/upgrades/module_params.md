---
title: Params adjustments
sidebar_position: 3
---

# Params adjustments <!-- omit in toc -->

## Parameters and the DAO

Pocket Network utilizes an offchain governance mechanism that enables the community to vote on proposals. Once a proposal passes, the DAO can adjust the parameters necessary for the protocol's operation.

- [Parameters and the DAO](#parameters-and-the-dao)
- [Examples](#examples)
  - [Block Size Change](#block-size-change)

## Access Control

// TODO_DOCUMENT(@bryanchriswhite) tl;dr, authz.

## Examples

### Block Size Change

Similar to how internal parameters can be adjusted using [Adding params](../../develop/developer_guide/adding_params.md), the DAO can submit changes to other modules. For example, here's a transaction that will increase the block size (a parameter in the `consensus` module):

```json
{
  "body": {
    "messages": [
      {
        "@type": "/cosmos.consensus.v1.MsgUpdateParams",
        "authority": "pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t",
        "abci": {},
        "block": {
          "max_bytes": "66060288",
          "max_gas": "-1"
        },
        "evidence": {
          "max_age_duration": "48h0m0s",
          "max_age_num_blocks": "100000",
          "max_bytes": "1048576"
        },
        "validator": {
          "pub_key_types": ["ed25519"]
        }
      }
    ]
  }
}
```

:::warning
Important: When submitting changes, you must include all parameters, even if you're only modifying one.
:::

To check the current consensus parameters (before and after the change), use this command:

```bash
poktrolld query consensus params
```

Before the upgrade:

```yaml
params:
  block:
    max_bytes: "22020096"
  # ... the rest of the response
```

To submit the transaction that increases the block size:

```bash
poktrolld tx authz exec tools/scripts/params/consensus_increase_block_size.json --from pnf --yes
```

After the upgrade:

```yaml
params:
  block:
    max_bytes: "66060288"
  # ... the rest of the response
```
