---
title: Diagrams
sidebar_position: 1
---

:::warning TODO_UPNEXT(@bryanchriswhite,#1034)
Expand migration docs and re-organize this page.

    All migration documents can be found on notion [here](https://www.notion.so/buildwithgrove/Morse-to-Shannon-Migration-173a36edfff6809cb1cbe10827c040de?pvs=4).

:::

## Table of Contents <!-- omit in toc -->

- [Complete E2E User Sequence](#complete-e2e-user-sequence)
- [Onchain Actors \& Messages](#onchain-actors--messages)
- [Shannon Data Structures to Represent Morse State](#shannon-data-structures-to-represent-morse-state)

## Complete E2E User Sequence

```mermaid
sequenceDiagram
    autonumber

    actor A as Authority (Foundation)
    actor MA as Morse Account Holders
    participant MN as Morse Network
    participant SN as Shannon Network

    loop (Re-)Generate morse_account_state.json

        A ->>+ MN: $ pocket util export-geneis-for-reset
        MN ->>- A: return morse_state_export.json

        A ->> A: $ poktrolld migrate collect-morse-accounts
        note over A: morse_account_state.json generated

        A ->>+ MA: distribute for verification <br> morse_account_state.json

        opt Morse Stakeholders optionally do local verification
            MA ->>+ MN: $ pocket util export-geneis-for-reset
            MN ->>- MA: return for local verification <br> morse_state_export.json

            MA ->> MA: $ poktrolld migrate collect-morse-accounts
            note over MA: morse_account_state.json generated
            MA ->> MA: manual comparison of <br> morse_account_state.json hashes

            MA ->>- A: ** (off-chain feedback) **
        end

    end

    A ->>+ SN: upload morse state<br/>(MsgCreateMorseAccountState)
    SN ->> SN: verify morse_account_state_hash field
    SN -->- A: valid / invalid

    MA ->> SN: $ poktrolld migration claim-morse-pokt<br/>claim morse POKT<br/>(MsgClaimMorsePOKT)
```

## Onchain Actors & Messages

```mermaid
flowchart
    m1[MsgCreateMorseAccountClaim]:::account
    m2[MsgCreateMorseSupplierClaim]:::supplier
    m3[MsgCreateMorseApplicationClaim]:::application
    m4[MsgCreateMorseGatewayClaim]:::gateway

    subgraph MigrationKeeper
    h1([CreateMorseAccountClaim]):::account
    h2([CreateMorseSupplierClaim]):::supplier
    h3([CreateMorseApplicationClaim]):::application
    h4([CreateMorseGatewayClaim]):::gateway
    %% ms[[MorseAccountState]]
    ac[[MorseAccountClaims]]:::general
    end

    h1 --"Morse Account Claim Creation<br/>(ensure not previously claimed)"--> ac
    h2 --"Morse Account Claim Creation<br/>(ensure not previously claimed)"--> ac
    h3 --"Morse Account Claim Creation<br/>(ensure not previously claimed)"--> ac
    h4 --"Morse Account Claim Creation<br/>(ensure not previously claimed)"--> ac

    %% h1 --"ensure claim is valid"--> ms
    %% h2 --"ensure claim is valid"--> ms
    %% h3 --"ensure claim is valid"--> ms
    %% h4 --"ensure claim is valid"--> ms

    m1 --> h1
    m2 --> h2
    m3 --> h3
    m4 --> h4

    subgraph BankKeeper
    bk1[[Balances]]:::general
    end

    subgraph SupplierKeeper
    sk1[["Suppliers"]]:::supplier
    end

    subgraph ApplicationKeeper
    ak1[["Applications"]]:::application
    end

    subgraph GatewayKeeper
    gk1[["Gateways"]]:::gateway
    end

    h1 --"Mint Balance"----> bk1
    h2 --"Mint Supplier Stake &<br/>Non-staked Balance"--> bk1
    h2 --"Stake Supplier"---> sk1
    h3 --"Mint Application Stake &<br/>Non-staked Balance"--> bk1
    h3 --"Stake Application"---> ak1
    h4 --"Mint Gateway Stake &<br/>Non-staked Balance"--> bk1
    h4 --"Stake Gateway"---> gk1

    classDef account fill:#90EE90,color:#000
    classDef supplier fill:#FFA500,color:#000
    classDef application fill:#FFB6C6,color:#000
    classDef gateway fill:#FF0000,color:#000
    classDef general fill:#87CEEB,color:#000
```

## Shannon Data Structures to Represent Morse State

```mermaid
---
title: MorseStateExport --> MorseAccountState Transform
---

flowchart

subgraph OffC[Off-Chain]
    MorseStateExport --> MorseAppState
    MorseAppState --> MorseApplications
    MorseAppState --> MorseAuth
    MorseAppState --> MorsePos
    MorseApplications --> MorseApplication
    MorseAuth --> MorseAuthAccount
    MorseAuthAccount --> MAOff[MorseAccount]
    MorsePos --> MorseValidator
end


subgraph OnC[On-Chain]
    MorseAccountState
    MorseAccountState --> MAOn
    MAOn[MorseAccount]
end

MAOff -.add exported account balance..-> MAOn
MorseValidator -.add exported stake to account balance.-> MAOn
MorseApplication -.add exported stake to account balance.-> MAOn
```

```mermaid
---
title: MorseStateExport Structure(s)
---

classDiagram

class MorseStateExport {
    app_hash: string
    app_state: MorseAppState
}
MorseStateExport --* MorseAppState

class MorseAppState {
    application: MorseApplications
    auth: MorseAuth
    pos: MorsePos
}
MorseAppState --* MorseApplications
MorseAppState --* MorseAuth
MorseAppState --* MorsePos

class MorseApplications {
  applications: []MorseApplication
}

class MorseAuth {
  accounts: []MorseAuthAccount
}
MorseAuth --* MorseAuthAccount

class MorseAuthAccount {
    type: string
    value: MorseAccount
}
MorseAuthAccount --* MorseAccount

class MorsePos {
    validators: []MorseValidator
}
MorsePos --* MorseValidator

class MorseValidator {
    address: bytes
    public_key: bytes
    jailed: bool
    status: int32
    staked_tokens: string
}

class MorseApplication {
    address: bytes
    public_key: bytes
    jailed: bool
    status: int32
    staked_tokens: string
}
MorseApplications --* MorseApplication

class MorseAccount {
    address: string
    pub_key: MorsePublicKey
    coins: []cosmostypes.Coin
}
MorseAccount --* MorsePublicKey

class MorsePublicKey {
    value crypto/ed25519.PublicKey
}
```

```mermaid
---
title: MsgCreateMorseAccountState Structure(s)
---

classDiagram

class MsgCreateMorseAccountState {
    authority: string
    morse_account_state: MorseAccountState
    morse_account_state_hash: bytes
}
MsgCreateMorseAccountState --* MorseAccountState

class MorseAccountState {
    accounts: []MorseAccount
    GetHash(): []bytes
}
MorseAccountState --* MorseAccount

class MorseAccount {
    address: string
    pub_key: MorsePublicKey
    coins: []cosmostypes.Coin
}
MorseAccount --* MorsePublicKey

class MorsePublicKey {
    value crypto/ed25519.PublicKey
}
```
