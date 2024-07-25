---
title: Params adjustments
sidebar_position: 3
---

# Params adjustments <!-- omit in toc -->


## Parameters and the DAO

Pocket Network has an off-chain governance mechanism that allows the community to vote on proposals. After the proposal
is passed, DAO can adjust the parameters necessary for the protocol's operation.

- [Parameters and the DAO](#parameters-and-the-dao)
- [Examples](#examples)
  - [Block size change](#block-size-change)

## Examples

### Block size change

Similar to how the internal parameters can be adjusted using [Adding params](../../develop/developer_guide/adding_params.md), DAO can submit changes to other modules. For example, here is a transaction that will increase the block size (which is
a parameter in the `consensus` module):

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
          "pub_key_types": [
            "ed25519"
          ]
        }
      }
    ]
  }
}
```

Note that is is important to pass all the parameters, even if you are only changing one.

You can check the current consensus parameters (before and after) using this command:

```bash
poktrolld query consensus params
```

Before upgrade:
```yaml
params:
  block:
    max_bytes: "22020096"
  # ... the rest of the response  
```

Submitting the transaction to increase the block size:
```bash
poktrolld tx authz exec tools/scripts/params/consensus_increase_block_size.json --from pnf --yes
```

After upgrade:
```yaml
params:
  block:
    max_bytes: "66060288"
  # ... the rest of the response  
```