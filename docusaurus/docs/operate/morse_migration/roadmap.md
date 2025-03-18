---
title: Migration Roadmap
sidebar_position: 1
---

The `Morse` to `Shannon` migration is not a one day cutover. It is a process that
will span multiple weeks involving many stakeholders. It includes,
but is not limited to things like Genesis, Preparation, Onboarding, Cutover, etc...

The following diagram provides a high level overview of the migration process:

```mermaid
timeline
    title Shannon MainNet Launch Strategy

    section Shadow Genesis

      Shannon Block Genesis
        : Grove prepares minimal genesis file
        : Contains 1 Account <br> - <br> PNF w/ 5% of <br> Morse POKT <br> - <br> Bootstrap Allocation
        : Contains 1 Validator <br> - <br> Grove Validator w/ minimal stake
        : Inflation & emission Param Values <br> - <br> Set to Zero
        : Other Param Values <br> - <br> Not a blocker & can be updated later

      Shannon Relay Genesis
        : PNF -> Grove bootstrap funding
        : Grove creates the first Service for ETH MainNet
        : Grove stakes the first Application, Gateway & Supplier for ETH MainNet
        : Grove beings shadowing some traffic to Shannon MainNet
        : Manual E2E monitoring, observation, validation, etc...

    section Ecosystem Onboarding

      Internal Ecosystem Onboarding
        : PNF sends bootstrap funds onchain to enable al of the efforts below
        : Early adopters start run Full MainNet Nodes
        : Community deploys Block Explorers & Faucets
        : Identify the first ~6 Services <br> (i.e. Chains)
        : Identify the first ~4 Suppliers <br> (i.e. RelayMiners)
        : Identify the first ~2 Gateways <br> (i.e. Applications)

      Validator Strategy
        : Identify 1-3 Morse Pocket Network Validators interested in "Early Adopter Participation"
        : Identify 1-3 Cosmos Validators interested in "Early Adopter Participation"
        : Submit Gov Tx to increase the number of allowed Validator
        : Increase Validator distribution & diversity
        : CEXs deploy full nodes
        : DAO distributes to white-glove partners

      External Ecosystem Alignment & Kickoff
        : Kickoff IBC Bridging & Integration (Cosmos Hub, Layer, Noble)
        : Kickoff EVM Bridging & Integration (Axelar)
        : Kickoff Restaking & AVS Conversations (EigenLayer, Babylon)

    section Migration & Cutover

      Preparation
        : CEX Preparation <br> - <br> Provide RPC Endpoint <br> - <br> OR <br> - <br> Support Full Node Onboarding
        : Supplier Onboarding <br> - <br> Onboard Key Suppliers to support all Network traffic
        : Parameter R&D <br> - Prepare MainNet Governance params
        : Marketing & Comms Alignment w/ various stakeholders
        : Cosmos Metadata <br> - <br> Merge PRs Cosmos Tooling <br> (Keplr, Cosmos Registry)
        : Grove Load Test <br> - <br> 10B Relay Load Test <br> Mimic TestNet Results

      Judgement Day
        : CEXs <br> DROP support for Morse
        : Export Snapshot from Morse
        : Manually remove the <br> "Bootstrap Allocation" <br> from PNF's Account
        : Import Morse Snapshot into Shannon <br> (ImportMorseAccounts)
        : Stakeholders can begin claiming Morse POKT <br> (ClaimMorseAccount, ClaimMorseSupplier, ClaimMorseApplication)
        : CEXs <br> ADD support for Shannon

      Migration Stabalization
        : PNF Updates All Gov Params <br> - <br> Align Inflation with Morse as a starting point
        : Relay Migration <br> - <br> Grove and other Gateways migrate all traffic to Shannon
        : Token Support <br> - <br> Handle incoming support requests w.r.t token migration
        : Network Stability <br> - <br> Monitor network health & stability (relays, QoS, etc)
        : Tooling <br> - <br> Update docs & tooling based on incoming requests

    section Network Activation & Maturation

        Ecosystem Alignment
            : Continued Marketing & Comms
            : Make the validator set permissionless so anyone can onboard
            : Close the loop on IBC Integration
            : Close the loop on EVM Integration
            : DEX Integration
            : Continue effort on restaking integrations (Babylon & EigenLayer)

        Future Work
            : Outside of scope of this document
```

## Static Image

You can use [mermaid.live](https://mermaid.live/) to copy, paste, edit and zoom in on the source code for the diagram above.

Alternatively, you can use the following static image:

![Image](https://github.com/user-attachments/assets/7a2c5406-7c03-4778-aab1-85bdbdbfffb2)
