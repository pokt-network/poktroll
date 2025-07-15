---
title: Global Mint TLM
sidebar_position: 5
---

_tl;dr Distribute newly minted (net new) coins on a per claim basis to all involved stakeholders._

- [Token Distribution (Global Inflation)](#token-distribution-global-inflation)
  - [Example Distribution](#example-distribution)
- [TLM: Global Mint Reimbursement Request (GMRR)](#tlm-global-mint-reimbursement-request-gmrr)
  - [Self Dealing Attack](#self-dealing-attack)
  - [Reimbursement Request Philosophy](#reimbursement-request-philosophy)
  - [Reimbursement Request Design](#reimbursement-request-design)
- [FAQ](#faq)

## Token Distribution (Global Inflation)

The `Global Mint` TLM is, _theoretically_, going to reach `zero` when the network
reaches maturity in the far future.

On a per claim basis, the network mints new tokens based on the amount of work
claimed. The newly minted tokens are distributed to the DAO, Service Owner, Application,
Supplier and its Revenue Shareholders based on the values of various governance params.

### Example Distribution

For example, assuming the following tokenomic module params:

```json
{
  "params": {
    "mint_allocation_percentages": {
      "dao": 0.1,
      "proposer": 0.1,
      "supplier": 0.7,
      "source_owner": 0.1,
      "application": 0
    },
    "dao_reward_address": "pokt10...",
    "global_inflation_per_claim": 0.2
  }
}
```

The newley minuted tokens would be distributed as follows:

```mermaid
graph TD
    %% Set default styling for all nodes
    classDef default fill:#f9f9f9,stroke:#333,stroke-width:1px,color:black;

    uPOKT["**🧾 Session.Cost**<br/>(uPOKT)"]:::keyClass

    subgraph GlobalInflation["🌍 Global Inflation Calculation"]
        direction TB
        GIP["**chain.GlobalInflationPerClaim**<br/>(0.2)"]:::default
        InflationAmount["**💰 session.Inflation**<br/>(uPOKT × 0.2)"]:::default

        GIP --> InflationAmount
        uPOKT --> InflationAmount
    end

    subgraph TokenMinting["⚡ Token Minting & Distribution"]
        direction TB

        TotalMint["**🏭 session.MintBurnInflate**<br/>(Session Cost + Inflation)"]:::default

        AppWallet[("📱 Application Wallet<br/>(User Controlled)")]:::configClass
        SupplierWallet[("🏗️ Supplier Wallet<br/>(Operator Controlled)")]:::supplierClass
        DAOAddress[("🏛️ DAO Treasury<br/>pokt10...")]:::daoClass
        ProposerWallet[("👨‍⚖️ Block Proposer<br/>(Validator Wallet)")]:::proposerClass
        SourceOwnerWallet[("🛎 Service Owner Wallet<br/>(Service Controlled)")]:::sourceClass
    end

    InflationAmount --> TotalMint

    TotalMint -->|"🔥 Session.Cost"| AppWallet
    TotalMint -->|"💰 Session.Cost + <br/> 💰 0.7 * Session.Inflation"| SupplierWallet
    TotalMint -->|"💰 0.1 * Session.Inflation"| DAOAddress
    TotalMint -->|"💰 0.1 * Session.Inflation"| ProposerWallet
    TotalMint -->|"💰 0.1 * Session.Inflation"| SourceOwnerWallet

    %% Define custom classes with specified colors
    classDef userClass fill:#f0f0f0,stroke:#333,stroke-width:2px,color:black;
    classDef gatewayClass fill:#e8f5e8,stroke:#4caf50,stroke-width:2px,color:black;
    classDef pocketdClass fill:#fff3e0,stroke:#ff8f00,stroke-width:2px,color:black;
    classDef blockchainClass fill:#e3f2fd,stroke:#2196f3,stroke-width:2px,color:black;
    classDef supplierClass fill:#fff3e0,stroke:#ff9800,stroke-width:2px,color:black;
    classDef keyClass fill:#ffebee,stroke:#d32f2f,stroke-width:1px,color:black;
    classDef configClass fill:#f3e5f5,stroke:#7b1fa2,stroke-width:1px,color:black;
    classDef dbClass fill:#e0f2f1,stroke:#00695c,stroke-width:1px,color:black;
    classDef daoClass fill:#e8eaf6,stroke:#3f51b5,stroke-width:1px,color:black;
    classDef proposerClass fill:#f1f8e9,stroke:#689f38,stroke-width:1px,color:black;
    classDef sourceClass fill:#fce4ec,stroke:#c2185b,stroke-width:1px,color:black;

    %% Apply classes to subgraphs
    class GlobalInflation pocketdClass
    class TokenMinting blockchainClass
```

## TLM: Global Mint Reimbursement Request (GMRR)

_tl;dr Prevent self-dealing by over-charging applications, sending the excess to the DAO/PNF, and emitting an event as a reimbursement request._

### Self Dealing Attack

A self-dealing attack is when an application leverages the inflationary nature of the
tokenomics to increase its balance by sending spam traffic.

- Above the `Inflation` note, the number of tokens in circulation remains constant.
- After the `Inflation` note, the number of tokens in circulation increases.

**If the individual managing the Application/Gateway is the same one who is managing
the Supplier and/or Service Owner, they could mint an unbounded number of new tokens
for themselves by sending fake traffic. This is shown in red.**

```mermaid
---
title: "Self Dealing Attack"
---
sequenceDiagram
    actor U as End User
    participant AG as Application | Gateway
    participant S as Supplier
    participant P as Protocol
    participant SO as Service Owner
    participant DAO as DAO
    participant BP as Block Proposer

    loop "provide service throughout session"
        U ->> +AG: RPC Request
        AG ->> +S: POKT Relay Request
        S ->> -AG: POKT Relay Response
        AG ->> -U: RPC Response
    end

    critical "settle session accounting"
        AG -->> +P: Pay POKT (based on work done)
        P -->> -S: Receive POKT (based on work done)
        note over AG,P: Inflation: Mint new POKT
        rect rgb(247, 13, 26)
            P -->> S: Send rewards (% of mint)
            P -->> SO: Send mint rewards (% of mint)
        end
        P -->> DAO: Send mint rewards (% of mint)
        P -->> BP: Send mint rewards (% of mint)
    end
```

### Reimbursement Request Philosophy

_Solving the above problem is non-trivial_.

See the [resources](1_resources.md) for more information on the long-term game-theoretic solutions.

In the meantime, the interim manual approach described below is a stepping stone
do things that don't scale in the short term, but can be easily automated, while
enabling permissionless demand and dissuading self-dealing attacks.

### Reimbursement Request Design

This TLM is a dependency of the Global Mint TLM; i.e., it **MUST** be active ONLY IF Global Mint is active.

This TLM can, **theoretically**, be removed if self-dealing attacks are not a concern,
or if the global mint per claim governance parameter is set to zero.

The goal of the TLM is supplement the Global Mint TLM such that:

1. The application is overcharged by the inflation amount in `TLM: Global Mint`.
2. The application must **"show face"** in front of the DAO/PNF to request reimbursement.
3. PNF needs to **manually approve** the reimbursement request.

**While this is not perfect, it follows on the **[Deterrence Theory](<https://en.wikipedia.org/wiki/Deterrence_(penology)>)** that
the increased risk of punishment will dissuade bad actors.**

_NOTE: A side effect of this TLM is creating additional buy pressure of the token as Applications
and Gateways will be responsible for frequently "topping up" their balances and app stakes._

```mermaid
---
title: "Token Logic Module: Global Mint Reimbursement Request"
---
flowchart TD
    SA(["Settlement Amount (SA)"])
    PCI(["Per Claim Global Inflation (PCGI)"])
    ARRE{{Application Reimbursement <br> Request Event}}
    TM2["Tokenomics Module"]

    subgraph TLMGM["TLM: Global Mint"]
        IA(["Inflation Mint Coin (IA)"])
        ID["Inflation Distribution <br> (see TLM above for details)"]
        GP(["Governance Params"])

        subgraph TO[Tokenomics Operations]
            TM[[Tokenomics Module]]
            IA("IA = SA * PCGI <br> (IA: Inflation Amount)")
            TM  --> |"💲 MINT IA"|IA
        end

        %% Mint Inputs
        SA --> TO
        PCI --> TO

        %% Distribute Inflation
        TO --> |"Distribute Inflation (IA) <br> ⬆️ INCREASE Balances"| ID
        GP --> ID
    end

    subgraph AO[Application Operations]
        AM[[Application Module]]
        AK[(Application Keeper)]
        AA[Application Address]

        %% Reimbursement Request Actions
        AM -.- AK
        AM -. ⬇️ REDUCE Stake by IA .-> AA
    end


    %% Reimbursement Request Logic
    TO ---> |Prevent Self Dealing <br> 🤝 HOLD IA| AO
    AO -.-> |Emit Event| ARRE
    AO --> |"⬆️ INCREASE Module Balance (IA)"| TM2

    classDef module fill:#f9f,color: #333,stroke:#333,stroke-width:2px;
    classDef address fill:#bbf,color: #333,stroke:#333,stroke-width:2px;
    classDef govparam fill:#eba69a,color: #333,stroke:#333,stroke-width:2px;
    classDef event fill:#e8b761,color: #333,stroke:#333,stroke-width:2px;

    class TM,AM,TM2 module;
    class PCI,GP govparam;
    class PRA,AA,DAO address;
    class ARRE event;
```

Later, PNF, on behalf of the DAO, will review the reimbursement requests and approve them.

```mermaid
---
title: "Offchain Reimbursement Request Flow"
---
sequenceDiagram
    participant PNF as Pocket Network Foundation
    participant BS as Blockchain State
    participant T as Tokenomics Module
    participant Apps as Applications 1..N

    PNF ->> +BS: Get All Reimbursement Requests
    BS ->> -PNF: List of Reimbursement Requests
    PNF ->> PNF: Review Reimbursement Requests
    loop "for each request"
        alt "Approve"
            PNF ->> T: Reimburse Application Funds
            T ->> Apps: Send Reimbursement
        else "Reject"
            note over PNF, Apps: PNF maintains funds
        end
    end
```

## FAQ

### Are Applications responsible for endorsing/covering the whole global mint amount? <!-- omit in toc -->

_tl;dr Yes, for the first version._

The application `PAYS` the supplier for work done (i.e. Mint=Burn).
The application `GETS REIMBURSED` for the inflation (i.e. Global Mint).

This will require staked Applications (sovereign or those managed by Gateways) to periodically
"top up" their balances to cover not only the onchain costs/burn, but also the inflation
until it is reimbursed by the DAO/PNF.

### Will there be onchain enforcement of how Applications get reimbursed? <!-- omit in toc -->

_tl;dr Unfortunately, no._

The Applications will indeed have to trust the DAO/PNF to reimburse them.
The following is an example of the approach PNF could take.

1. Assume Application staking by Gateways is permissionless and done.
2. Applications pay onchain for costs and inflation
3. PNF KYCs Gateways who seek reimbursement.
4. Gateways that don't go through the KYC process cover the cost of inflation
   out of pocket.
5. A script that retrieves onchain reimbursement requests will be written that
   automatically send funds to previously KYCed gateways
6. The script above, and the trust that it'll be maintained, updated and executed
   relies in the Gateways' trust in the PNF.

This is similar, in spirit, but still an improvement on top of the trust
between Gateways and PNF in Morse today in order to:

- Get access to the limited supply of Gateway keys
- Gateways paying the onchain burn manually

### How does this solution scale for Sovereign Applications? <!-- omit in toc -->

Sovereign Applications are no different than Gateway Applications in this respect.
They are smaller and a much less common use case, but will have to follow the same
reimbursement process described above.

_Read more about about their differences and similarities [here](../primitives/gateways.md)._

### What kind of resources are needed to scale and automate reimbursement? <!-- omit in toc -->

This will be a combination of onchain and offchain resources (EventReader, TxSubmission, Accounting, etc...). In particular:

- **Onchain**: load testing will show if events take up too much onchain space. This is unlikely to be an issue relative to proofs.
- **Offchain**: PNF Directors are aware and approve of the operational overhead this will require. This will require some offchain scripting to automate the process.
