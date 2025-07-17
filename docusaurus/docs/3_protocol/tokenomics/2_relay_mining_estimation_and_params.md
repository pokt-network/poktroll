---
title: Relay Mining Estimation & Parameters
sidebar_position: 2
---

A gentle high-level introduction to token minting and burning, intended to understand
the different variables involved and how they interact.

- [Mathematic Representation](#mathematic-representation)
- [Relay Mining Parameters \& Function](#relay-mining-parameters--function)
- [Claim \& Relay Estimation Flow](#claim--relay-estimation-flow)
- [💰 Example with Numbers](#-example-with-numbers)
- [FAQ](#faq)
  - [Why do we need relay mining difficulty?](#why-do-we-need-relay-mining-difficulty)
  - [How do we prove the claim?](#how-do-we-prove-the-claim)
  - [Why is every relay the same number of compute units?](#why-is-every-relay-the-same-number-of-compute-units)
  - [How does rate limiting work?](#how-does-rate-limiting-work)
  - [How does burning work?](#how-does-burning-work)

## Mathematic Representation

```math
\begin{aligned}
\text{Claim.NumRelays} &= \text{scaleDown}(\text{RelayMiningDifficulty}, \text{ActualNumberOfRelays}) \\
\text{ClaimedComputeUnits} &= \text{Claim.NumRelays} \times \text{ComputeUnitsPerRelay} \\
\text{EstimatedOffchainComputeUnits} &= \text{scaleUp}(\text{RelayMiningDifficulty}, \text{ClaimedComputeUnits}) \\
u\text{POKT} &= \frac{\text{EstimatedOffchainComputeUnits} \times \text{ComputeUnitsToTokenMultiplier}}{\text{ComputeUnitCostGranularity}}
\end{aligned}
```

## Relay Mining Parameters & Function

| Parameter                       | Type               | Scope            | Controller                         | Description                                                                               |
| ------------------------------- | ------------------ | ---------------- | ---------------------------------- | ----------------------------------------------------------------------------------------- |
| `RelayMiningDifficulty`         | Dynamic Parameter  | Service Specific | Onchain protocol business logic    | The probability that a relay is reward applicable                                         |
| `ComputeUnitsPerRelay`          | Static Parameter   | Service Specific | Service Owner                      | Number of compute units each reward applicable relay accounts for                         |
| `ComputeUnitsToTokenMultiplier` | Static Parameter   | Network Wide     | Network Authority                  | Number of onchain tokens minted/burnt per compute unit                                    |
| `ComputeUnitCostGranularity`    | Static Parameter   | Network Wide     | Network Authority                  | Enable more granular calculations for the cost of a single relay (i.e. less than 1 uPOKT) |
| `scaleDown`                     | Offchain procedure | Service Specific | RelayMiner & Onchain Params        | A methodology to ensure RelayMiner scalability                                            |
| `scaleUp`                       | Onchain procedure  | Service Specific | Validator & Onchain Business Logic | A methodology to ensure fair token distribution                                           |

## Claim & Relay Estimation Flow

The end-to-end flow can be split into 4 key steps (steps 1-3 capture in the diagram below):

1. **Tree Construction**: Converts actual offchain number of relays to the probabilistic number of reward applicable relays
2. **Claim Creation**: Sums up the total number of reward applicable relays in the tree into a single claim
3. **Claim Settlement**: Estimates the total number of consumed compute units based on the claim and service parameters
4. **Token Distribution**: Converts the estimated compute units into uPOKT and distributes it to relevant stakeholders

```mermaid
graph TD
    %% Set default styling for all nodes
    classDef default fill:#f9f9f9,stroke:#333,stroke-width:1px,color:black;

    User("👨‍💻 User / Developer"):::userClass
    ServiceOwner("💁 Service Owner"):::userClass

    subgraph HO["👷 Hardware Operator"]
        direction LR
        RelayMiner[["⛏️ RelayMiner<br/>(Co-processor)"]]:::supplierClass
        Tree[("🌲 Sparse Merkle Sum Trie<br/> **Root Sum: Num Relays**<br/>(Reward Applicable Relays)")]:::dbClass
        RelayMiner -.- | **🎲 Probabilistic Number** <br/> of Relays Serviced<br/>|Tree
    end
    HO:::userClass

    User <-->|"**🔢 Actual Number**<br/> of Relays Serviced"| HO
    Tree -->|"Claim Creation"| Blockchain
    User -.- App
    ServiceOwner -.- Service
    HO -.- Supplier

    subgraph Blockchain["🌀 Pocket Network"]
        direction TB

        App[("📱 Application<br/>(User Owned)")]:::configClass
        Supplier[("🏗️ Supplier<br/>(Operator Owned)")]:::configClass
        Service[("🛎 Service<br/>(Service Owned)<br/>")]:::configClass

        subgraph Calculation["Tokenomics Calculation"]
            CUPR["**Service.ComputeUnits<br/>PerRelay**"]:::default
            RMD["**Service.RelayMining<br/>Difficulty**"]:::default
            NumRelays["**Claim.NumRelays**<br/>(Reward Applicable Relays)"]:::default

            NumRelays --> EstimatedCU
            CUPR --> EstimatedCU
            RMD --> EstimatedCU

            uPOKT["**🧾 Session.Cost**<br/>(uPOKT)"]:::default
            EstimatedCU["**🧮 Total Estimated** <br/>Compute Units"]:::default --> uPOKT
        end
    end

    %% Define custom classes with specified colors
    classDef userClass fill:#f0f0f0,stroke:#333,stroke-width:2px,color:black;
    classDef gatewayClass fill:#e8f5e8,stroke:#4caf50,stroke-width:2px,color:black;
    classDef pocketdClass fill:#fff3e0,stroke:#ff8f00,stroke-width:2px,color:black;
    classDef blockchainClass fill:#e3f2fd,stroke:#2196f3,stroke-width:2px,color:black;
    classDef supplierClass fill:#fff3e0,stroke:#ff9800,stroke-width:2px,color:black;
    classDef keyClass fill:#ffebee,stroke:#d32f2f,stroke-width:1px,color:black;
    classDef configClass fill:#f3e5f5,stroke:#7b1fa2,stroke-width:1px,color:black;
    classDef dbClass fill:#e0f2f1,stroke:#00695c,stroke-width:1px,color:black;

    %% Apply classes to subgraphs
    class Blockchain blockchainClass
    class Calculation pocketdClass
```

## 💰 Example with Numbers

Assume the following Offchain market driven numbers:

- **POKT price**: $0.1/POKT
- **Market rate**: $5M for 1M relays
- **Session**: Num actual offchain relays between App (User) & Supplier (Operator)

| Num Relays     | Description                            | RelayMiningDifficulty (RMD) | ComputeUnitsPerRelay (CUPR) | ComputeUnitsToTokenMultiplier (CUTTM) | ComputeUnitCostGranularity (CUCG) | Estimated Compute Units (CU)                       | uPOKT Result                                                                   | USD <br/> (at $0.10/POKT) |
| -------------- | -------------------------------------- | --------------------------- | --------------------------- | ------------------------------------- | --------------------------------- | -------------------------------------------------- | ------------------------------------------------------------------------------ | ------------------------- |
| **1,000,000**  | Baseline values                        | 1.0                         | 1.0                         | 50                                    | 1                                 | 1,000,000 x 1 x 1 <br/> = 1,000,000                | 1,000,000 × 50 / 1 <br/> = **50,000,000 uPOKT <br/> = 50 POKT**                | **$5.00**                 |
| **1,000,000**  | High multiplier <br/> High granularity | 1.0                         | 1.0                         | 50,000,000                            | 1e6                               | 1,000,000 x 1 x 1 <br/> = 1,000,000                | 1,000,000 × 50,000,000 / 1,000,000 <br/> = **50,000,000 uPOKT <br/>= 50 POKT** | **$5.00**                 |
| **1,000,000**  | High compute units per relay           | 1.0                         | 5.0                         | 50                                    | 1                                 | 1,000,000 x 1 x 5 <br/> = 5,000,000                | 5,000,000 × 50 / 1 <br/> = **250,000,000 uPOKT <br/> = 250 POKT**              | **$25.00**                |
| **10,000,000** | Adjusted relay mining difficulty       | 0.1                         | 1.0                         | 50                                    | 1                                 | 1,000,000 x <br/>(0.1 / 0.1) x 1 <br/> = 1,000,000 | 1,000,000 × 50 / 1 <br/> = **5,000,000 uPOKT <br/> = 5 POKT**                  | **$5.00**                 |

## FAQ

### Why do we need relay mining difficulty?

To be able to scale a single RelayMiner co-processor to handle billions of relays while being resource efficient.

### How do we prove the claim?

Visit the [claim and proof lifecycle docs](../primitives/claim_and_proof_lifecycle.md) for more information.

### Why is every relay the same number of compute units?

We will handle variable compute units per relay in the future.

### How does rate limiting work?

Rate limiting is an optimistic non-interactive permissionless mechanism that uses a commit-and-reveal scheme with probabilistic guarantees, crypto-economic (dis)incentives, and onchain proofs to ensure that suppliers do not over-service applications.

### How does burning work?

Burning is a mechanism that puts funds in escrow, burns it after work is done, and puts optimistic limits in place whose work volume is proven onchain.
