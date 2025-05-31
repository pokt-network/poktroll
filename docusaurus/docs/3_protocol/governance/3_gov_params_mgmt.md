---
title: Governance Params Management
sidebar_position: 3
---


## Governance Parameters

### User Experience Questions

1. **What governance params are available?**
2. **How do I check the current governance param values?**
3. **How do I update a specific value?**

### Parameter Categories

#### Application Parameters

- **App Min Stake**: Minimum stake required for application participation
- **App Max Delegated Gateways**: Maximum number of gateways an application can delegate to

#### Cosmos Parameters

- Validator Parameters
- Consensus Parameters
- Staking Parameters
- Slashing Parameters

#### Service Module Parameters

- Service-specific configuration parameters

#### Tokenomics Parameters

- Parameters controlling token economics and reward calculations

#### Shared Parameters

- Cross-module shared configuration values

- Show how to query everything
- Show how to update everything

:::warning Authority only

This page is for Pocket Network Authority members only on how to update and manage onchain parameters.

It can be used by developers on LocalNet but can only be executed by the foundation on MainNet.

:::

Ran this

```bash
pkd_beta_query tokenomics params -o json | jq
```

Got this:

```json
{
  "params": {
    "mint_allocation_percentages": {
      "dao": 0.1,
      "proposer": 0.05,
      "supplier": 0.7,
      "source_owner": 0.15,
      "application": 0
    },
    "dao_reward_address": "pokt1f0c9y7mahf2ya8tymy8g4rr75ezh3pkklu4c3e",
    "global_inflation_per_claim": 0.1
  }
}
```

Got this:

```json
{
  "params": {
    "mint_allocation_percentages": {
      "dao": 0.1,
      "proposer": 0.1,
      "supplier": 0.2,
      "source_owner": 0.1,
      "application": 0.5
    },
    "dao_reward_address": "pokt1f0c9y7mahf2ya8tymy8g4rr75ezh3pkklu4c3e",
    "global_inflation_per_claim": 0.5
  }
}
```

Create this:

```json
{
  "body": {
    "messages": [
      {
        "@type": "/pocket.tokenomics.MsgUpdateParams",
        "authority": "pokt1f0c9y7mahf2ya8tymy8g4rr75ezh3pkklu4c3e",
        "params": {
          "mint_allocation_percentages": {
            "dao": "0.1",
            "proposer": "0.1",
            "supplier": "0.2",
            "source_owner": "0.1",
            "application": "0.5"
          },
          "dao_reward_address": "pokt1f0c9y7mahf2ya8tymy8g4rr75ezh3pkklu4c3e",
          "global_inflation_per_claim": "0.5"
        }
      }
    ]
  }
}
```

Put it here: `tools/scripts/params_templates/tokenomics_0_all_beta_test.json`

Executed like so

```bash
pkd_beta_tx authz exec tools/scripts/params_templates/tokenomics_0_all_beta_test.json --from pnf_beta
```

Future:

```bash
pkd tx authz exec tools/scripts/params_templates/tokenomics_0_all_beta_test.json --from pnf_beta --yes --network=beta
```

Check tx result:

```bash
pkd_beta_query tx --type=hash 36A6C6F46AA0CFA99053EF8C2D8C52BAB3C66612407FEEBBF4427E58EAA30102
```

Second attempt

```json
{
  "body": {
    "messages": [
      {
        "@type": "/pocket.tokenomics.MsgUpdateParams",
        "authority": "pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t",
        "params": {
          "mint_allocation_percentages": {
            "dao": "0.1",
            "proposer": "0.1",
            "supplier": "0.2",
            "source_owner": "0.1",
            "application": "0.5"
          },
          "dao_reward_address": "pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t",
          "global_inflation_per_claim": "0.5"
        }
      }
    ]
  }
}
```

```bash
pkd_beta_query tx --type=hash 63CB416C1FDB4FD1AAB8C539CD71EA33D002AAF67F2B66C5B68003B78C9E6B9C
```

```bash
pkd_beta_query tokenomics params -o json | jq
{
  "params": {
    "mint_allocation_percentages": {
      "dao": 0.1,
      "proposer": 0.1,
      "supplier": 0.2,
      "source_owner": 0.1,
      "application": 0.5
    },
    "dao_reward_address": "pokt10d07y265gmmuvt4z0w9aw880jnsr700j8yv32t",
    "global_inflation_per_claim": 0.5
  }
}
```
