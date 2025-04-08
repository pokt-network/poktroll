---
title: Module Authorizations
sidebar_position: 3
---

## Module Authorizations <!-- omit in toc -->

Pocket Network utilizes an onchain governance mechanism that enables the community to vote on proposals.

Once a proposal passes, or a decision by PNF is made on behalf of the DAO, the parameters updates are applied.

- [Access Control](#access-control)
  - [MainNet Authorizations](#mainnet-authorizations)
    - [`x/gov` Module Granter](#xgov-module-granter)
    - [`PNF` Account Grantee](#pnf-account-grantee)
- [Examples](#examples)
  - [Example: Adding a new Authorization](#example-adding-a-new-authorization)

## Access Control

### MainNet Authorizations

The list of authorizations enabled on MainNet genesis can be found at [pokt-network/pocket-network-genesis/tree/master/shannon/mainnet](https://github.com/pokt-network/pocket-network-genesis/tree/master/shannon/mainnet).

#### `x/gov` Module Account

The `x/gov` module account is deterministically tied to address `pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t`. This will be true for any "pocket" network; e.g., LocalNet, TestNet, MainNet, etc.

The `x/gov` module account is the default configured "authority" for all cosmos-sdk modules. Each module references its own configured "authority" when executing messages which require authorization (e.g. MsgUpdateParams messages).

No one has access to this address, but the grants it has provided to other accounts can be queried like so:

```bash
pocketd query authz grants-by-granter pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t --node https://shannon-grove-rpc.mainnet.poktroll.com
```

#### `PNF` Account Grantee

The authorizations which the `x/gov` module account has granted to the `PNF` account can be queried like so:

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

```bash
pocket tx authz grant \
  pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw \
  "pocket.migration.MsgUpdateParams" \
  --from pnf \
  --expiration "2500-01-01T00:00:00Z" \
  --chain-id pocket \
  --gas auto
```
