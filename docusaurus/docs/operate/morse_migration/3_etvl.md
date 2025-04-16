---
title: ETVL Overview
sidebar_position: 3
---

ETVL stands for Export -> Transform -> Validate -> Load.

## Table of Contents <!-- omit in toc -->

- [ETVL Overview](#etvl-overview)
- [ETVL Technical Design Considerations \& Constraints](#etvl-technical-design-considerations--constraints)
- [ETVL High-Level Flow](#etvl-high-level-flow)

### ETVL Overview

Given that this migration involves representing the state of one network (Morse) in another (Shannon), and that the migration process is ongoing (i.e. not a re-genesis; see [constraints](#constraints)),
there is an opportunity to optimize the exported Morse state with respect to its (very long-term) impact on Shannon.

### ETVL Technical Design Considerations & Constraints

In order to streamline the migration process for end users, as well as expedite a high quality implementation, the following design considerations were applied:

1. **Re-use existing Morse tooling**:

   - Using the Morse CLI to export the canonical `MorseStateExport` from the Morse network (`pocket util export-genesis-for-reset`).
   - Using the Morse CLI to export (armored) Morse account keys for use with the Shannon claiming CLI (`pocketd migrate claim-...`).

2. **Facilitate offchain social consensus on MorseAccountState**:

   - Using social consensus and cryptographic hash verification
   - Offchain agreement (i.e. feedback loop) on the "canonical" `MorseAccountState`

3. Minimize Shannon onchain state bloat

   - Minimize the size & optimize performance of (Shannon) persisted onchain data
   - Transform (offchain) the `MorseStateExport` into a `MorseAccountState`
   - Persist minimal Morse account representations as individual `MorseClaimableAccount`s

### ETVL High-Level Flow

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
