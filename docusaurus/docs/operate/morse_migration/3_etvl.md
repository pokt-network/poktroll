---
title: ETVL Overview
sidebar_position: 3
---

**ETVL = Export → Transform → Validate → Load**

---

## Table of Contents <!-- omit in toc -->

- [What is ETVL?](#what-is-etvl)
- [Design Considerations \& Constraints](#design-considerations--constraints)
- [ETVL High-Level Flow](#etvl-high-level-flow)

---

## What is ETVL?

- ETVL is the process for migrating state from the Morse network to the Shannon network.
- This is **not** a full restart (not a re-genesis).
- Goal: Optimize the exported Morse state for long-term impact on Shannon.

---

## Design Considerations & Constraints

1. **Re-use Morse CLI tooling**

   - Export Morse state:

     ```bash
     pocket util export-genesis-for-reset ...
     ```

   - Export Morse account keys for Shannon claims:

     ```bash
     pocketd txmigrate claim-...
     ```

2. **Offchain social consensus**

   - Use cryptographic hash verification
   - Community agrees offchain on the canonical `MorseAccountState`

3. **Minimize Shannon onchain state bloat**

   - Keep onchain data small and fast
   - Transform Morse export into minimal `MorseClaimableAccount` objects
   - Only store what’s needed for claims

---

## ETVL High-Level Flow

```mermaid
  flowchart
    subgraph OncMorse[Morse On-Chain]
        MorseState
    end
    MorseState --> |<b>Export</b>| MorseStateExport
    subgraph OffC[Off-Chain]
        subgraph MorseStateExport
            tf[/<b>Transform</b>/]:::grey
            tf -.-> MorseAppState
            MorseAppState --> MorseApplications:::application
            MorseAppState --> MorseAuth:::account
            MorseAppState --> MorsePos:::supplier
            MorseApplications:::application --> MorseApplication:::application
            MorseAuth:::account --> MorseAuthAccount:::account
            MorseAuthAccount:::account --> MAccOff[MorseAccount]:::account
            MorsePos:::supplier --> MorseValidator:::supplier
        end
        subgraph MASOff[MorseAccountState]
            MAOff[MorseClaimableAccount]:::general
        end
        MAOnHA["MorseAccountStateHash (Authority Derived)"]:::general
        MAOnHU["MorseAccountStateHash (User Derived)"]:::general
        MAOnHU --> |"<b>Validate</b> (identical)"| MAOnHA
        MAOnHU --> |"<b>Validate</b> (SHA256)"| MASOff
    end
    subgraph OnC[Shannon On-Chain]
        subgraph MI[MsgImportMorseClaimableAccounts]
            subgraph MASOn[MorseAccountState]
                MAOn2[MorseClaimableAccount]:::general
            end
            MAOnH[MorseAccountStateHash]:::general
            MAOnH --> |"<b>Validate</b> (SHA256)"| MASOn
        end
        subgraph MM[Migration Module State]
            MAOn3[MorseClaimableAccount]:::general
            MAOn2 --> |"<b>Load</b> (Onchain Persistence)"| MAOn3
        end
    end
    MASOff -..-> |"<b>Load</b> (Authorized Tx)"| MASOn
    MAOnHA -.-> MAOnH
    MAccOff -.-> |Track exported <br> account unstaked balance| MAOff
    MorseValidator:::supplier -.-> |Track exported <br> Supplier stake| MAOff
    MorseApplication:::application -.-> |Track exported <br> Application stake| MAOff
    classDef account fill:#90EE90,color:#000
    classDef supplier fill:#FFA500,color:#000
    classDef application fill:#FFB6C6,color:#000
    classDef general fill:#87CEEB,color:#000
    classDef grey fill:#EAEAE7,color:#000
```
