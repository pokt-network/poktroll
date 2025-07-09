---
title: Mint=Burn TLM
sidebar_position: 5
---

_tl;dr The transfer of tokens from the applications to the suppliers based on the amount of work received and provided respectively._

The `Mint=Burn` TLM is, _theoretically_, the only TLM necessary once the network
reaches maturity in the far future.

The same number of tokens minted across all protocol participants is burned from
the **Application module**. The stake (in escrow) owned by the application which is
paying for work is reduced and the rewards are distributed according to the
`MintEqualsBurnClaimDistribution` parameters to suppliers, proposers, service source
owners, and the DAO.

```mermaid
---
title: "Token Logic Module: Mint=Burn"
---
flowchart TD
    SA(["Settlement Amount (SA)"])

    SA -- 💲 MINT SA coins --> TD
    SA -- 🔥 BURN SA coins--> AM

    subgraph TD[Token Distribution]
        MEC[MintEqualsBurnClaimDistribution]
        DAO_DIST[DAO Distribution]
        PROP_DIST[Proposer Distribution]
        SUPP_DIST[Supplier Distribution]
        SRC_DIST[Source Owner Distribution]

        MEC --> DAO_DIST
        MEC --> PROP_DIST
        MEC --> SUPP_DIST
        MEC --> SRC_DIST
    end

    subgraph SO[Supplier Operations]
        SM[[Supplier Module]]
        SD(Distribute Supplier Share)

        SUPP_DIST --> SM
        SM --> SD
        SD -->|"⬆️ INCREASE Balance <br> (% of supplier share)"| OA
        SD -->|"⬆️ INCREASE Balance <br> (% of supplier share)"| RSH1
        SD -->|"⬆️ INCREASE Balance <br> (% of supplier share)"| RSH2
        SD -->|"⬆️ INCREASE Balance <br> (% of supplier share)"| OPA

        subgraph RSA[Revenue Share Addresses]
            OA[Owner Address]
            OPA[Operator Address]
            RSH1[Revenue shareholder 1]
            RSH2[Revenue shareholder ...]
        end
    end

    subgraph AO[Application Operations]
        AM[[Application Module]]
        AK[(Application Keeper)]
        AA[Application Address]

        AM -.- AK
        AM -. ⬇️ REDUCE Stake by SA .-> AA
    end

    DAO_DIST -->|"⬆️ INCREASE Balance"| DAO_ADDR[DAO Address]
    PROP_DIST -->|"⬆️ INCREASE Balance"| PROP_ADDR[Proposer Address]
    SRC_DIST -->|"⬆️ INCREASE Balance"| SRC_ADDR[Source Owner Address]

    classDef module fill:#f9f,color: #333,stroke:#333,stroke-width:2px;
    classDef address fill:#bbf,color: #333,stroke:#333,stroke-width:2px;
    classDef distribution fill:#e8b761,color: #333,stroke:#333,stroke-width:2px;

    class SM,AM module;
    class RSH1,RSH2,OA,OPA,AA,DAO_ADDR,PROP_ADDR,SRC_ADDR address;
    class MEC,DAO_DIST,PROP_DIST,SUPP_DIST,SRC_DIST distribution;
```

## MintEqualsBurnClaimDistribution Parameters

The `MintEqualsBurnClaimDistribution` parameters control how the settlement amount is distributed across different network participants:

- **`dao`**: Percentage of settlement amount sent to the DAO reward address
- **`proposer`**: Percentage of settlement amount sent to the block proposer (validator)
- **`supplier`**: Percentage of settlement amount sent to suppliers (distributed among revenue shareholders)
- **`source_owner`**: Percentage of settlement amount sent to the service source owner
- **`application`**: Percentage of settlement amount that remains with the application (typically 0 for mint=burn)

These percentages must sum to 1.0 (100%) to ensure all settlement tokens are properly distributed.

### Default Distribution

The default distribution percentages are:
- **DAO**: 10% (0.1)
- **Proposer**: 5% (0.05)
- **Supplier**: 70% (0.7)
- **Source Owner**: 15% (0.15)
- **Application**: 0% (0.0)

### Parameter Governance

This parameter can be updated through governance proposals using the `MsgUpdateParams` message. All distribution percentages must be non-negative and sum to exactly 1.0.
