---
title: Params adjustments
sidebar_position: 3
---

## Param Adjustments <!-- omit in toc -->

:::warning TODO

TODO_IMPROVE(@olshansk): Refactor docs so authorization details and parameter
adjustments are in separate sections independent of protocol upgrades.

:::

## Parameters and the DAO

Pocket Network utilizes an offchain governance mechanism that enables the community to vote on proposals.

Once a proposal passes, or a decision by PNF is made on behalf of the DAO, PNF adjust the parameters necessary for the protocol's operation.

- [Parameters and the DAO](#parameters-and-the-dao)
- [Parametesr](#parametesr)
  - [Adding new Parameters](#adding-new-parameters)
  - [MsgUpdateParams vs MsgUpdateParam](#msgupdateparams-vs-msgupdateparam)
- [Access Control](#access-control)
  - [MainNet Authorizations](#mainnet-authorizations)
    - [`x/gov` Module Granter](#xgov-module-granter)
    - [`PNF` Account Grantee](#pnf-account-grantee)
- [Examples](#examples)
  - [Example: Adding a new Authorization](#example-adding-a-new-authorization)
  - [Example: Changing Num Suppliers Per Session](#example-changing-num-suppliers-per-session)
  - [Example: Block Size Change](#example-block-size-change)

## Parametesr

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
| **Poktroll**   | ❌ (Not available) | ✅ (Single param)  |

**Summary of Key Differences:**

- Cosmos SDK uses `MsgUpdateParams` (with **s**) which requires specifying **all** parameters in the module, even when modifying just one
- Poktroll implemented `MsgUpdateParam` (no **s**) allowing updates to **one** parameter at a time
- More details on the Cosmos SDK `MsgUpdateParams` can be found [here](https://hub.cosmos.network/main/governance/proposal-types/param-change).
- More details on poktroll's `MsgUpdateParam` can be found [here](https://github.com/search?q=repo%3Apokt-network%2Fpoktroll+%22message+MsgUpdateParam+%7B%22&type=code).

This distinction is critical when making on-chain parameter changes in either system.

## Access Control

### MainNet Authorizations

The list of authorizations enabled on MainNet genesis can be found at [pokt-network/pocket-network-genesis/tree/master/shannon/mainnet](https://github.com/pokt-network/pocket-network-genesis/tree/master/shannon/mainnet).

#### `x/gov` Module Granter

The `x/gov` module granter is tied to address `pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t`.

No one has access to this address, but the grants it has provided to other accounts can be queried like so:

```bash
pocketd query authz grants-by-granter pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t --node https://shannon-grove-rpc.mainnet.poktroll.com
```

#### `PNF` Account Grantee

The grants the `x/gov` module granter has provided to the `PNF` account can be queried like so:

```bash
pocketd query authz grants-by-grantee pokt1hv3xrylxvwd7hfv03j50ql0ttp3s5hqqelegmv --node https://shannon-grove-rpc.mainnet.poktroll.com
```

## Examples

### Example: Adding a new Authorization

In [this PR](https://github.com/pokt-network/poktroll/pull/1173/files), the following was added to `dao_genesis_authorizations.json`:

```json
  {
    "granter": "pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t",
    "grantee": "pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw",
    "authorization": {
      "@type": "cosmos.authz.v1beta1.GenericAuthorization",
      "msg": "pocket.migration.MsgUpdateParams"
    },
    "expiration": "2500-01-01T00:00:00Z"
  },
```

In order to enable it on an already deployed network, we need to submit the following transaction:

```json
pocket tx authz grant \
  pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw \
  "pocket.migration.MsgUpdateParams" \
  --from pnf \
  --expiration "2500-01-01T00:00:00Z" \
  --chain-id pocket \
  --gas auto
```

### Example: Changing Num Suppliers Per Session

To query the number of suppliers per session, use the following command:

```bash
pocketd query session params --node https://shannon-grove-rpc.mainnet.poktroll.com
```

To update the number of suppliers per session, you can use the following command:

```bash
{
  "body": {
    "messages": [
      {
        "@type": "/pocket.session.MsgUpdateParam",
        "authority": "pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t",
        "name": "num_suppliers_per_session",
        "as_uint64": "15"
      }
    ]
  }
}
```

Followed by

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
