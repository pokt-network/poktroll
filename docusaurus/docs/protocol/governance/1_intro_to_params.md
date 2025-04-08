---
title: Intro to Params
sidebar_position: 1
---

## Param Authorizations <!-- omit in toc -->

- [Parameters](#parameters)
  - [Adding new Parameters](#adding-new-parameters)
  - [MsgUpdateParams vs MsgUpdateParam](#msgupdateparams-vs-msgupdateparam)
- [Examples](#examples)
  - [Example: Changing Num Suppliers Per Session](#example-changing-num-suppliers-per-session)
  - [Example: Block Size Change](#example-block-size-change)

## Parameters

### Adding new Parameters

Visit [this page](../../develop/developer_guide/adding_params.md) for implementation details on how to add new parameters.

### MsgUpdateParams vs MsgUpdateParam

:::critical READ THIS
This is a critical distinction that can impact all onchain parameters.
:::

When submitting changes using `MsgUpdateParams` (note the **s**), you must specify
all parameters in the module even if just modifying one.

|                | MsgUpdateParams    | MsgUpdateParam     |
| -------------- | ------------------ | ------------------ |
| **Cosmos SDK** | ✅ (All params)    | ❌ (Not available) |
| **Pocket**   | ✅ (All params) | ✅ (Single param)  |

**Summary of Key Differences:**

- Cosmos SDK uses `MsgUpdateParams` (with **s**) which requires specifying **all** parameters in the module, even when modifying just one
- Poktroll implemented `MsgUpdateParam` (no **s**) allowing updates to **one** parameter at a time
- More details on the Cosmos SDK `MsgUpdateParams` can be found [here](https://hub.cosmos.network/main/governance/proposal-types/param-change).
- More details on poktroll's `MsgUpdateParam` can be found [here](https://github.com/search?q=repo%3Apokt-network%2Fpoktroll+%22message+MsgUpdateParam+%7B%22&type=code).

This distinction is critical when making on-chain parameter changes in either system.

## Examples

### Example: Changing Num Suppliers Per Session

To query the number of suppliers per session, use the following command:

```bash
pocketd query session params --node https://shannon-grove-rpc.mainnet.poktroll.com
```

To update the number of suppliers per session, you needs to create a new file with the transaction like so:

```bash
cat << 🚀 > /tmp/update_suppliers_per_session
{
  "body": {
    "messages": [
      {
        "@type": "/pocket.session.MsgUpdateParam",
        "authority": "pokt18808wvw0h4t450t06uvauny8lvscsxjfyua7vh",
        "name": "num_suppliers_per_session",
        "as_uint64": "15"
      }
    ]
  }
}
🚀
```

Followed by running:

```bash
pocketd tx authz exec /tmp/update_suppliers_per_session --from grove_mainnet_genesis --yes
```

### Example: Block Size Change

For example, here's a transaction that will increase the block size (a parameter in the `consensus` module):

:::note

For convenience, we have put it in `tools/scripts/params/consensus_block_size_6mb.json`.

:::

```json
{
  "body": {
    "messages": [
      {
        "@type": "/cosmos.consensus.v1.MsgUpdateParams",
        "authority": "pokt18808wvw0h4t450t06uvauny8lvscsxjfyua7vh",
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

To check the current consensus parameters (before and after the change), use this command:

```bash
pocketd query consensus params
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
pocketd tx authz exec tools/scripts/params/consensus_block_size_6mb.json --from pnf --yes
```

After the upgrade:

```yaml
params:
  block:
    max_bytes: "66060288"
  # ... the rest of the response
```
