---
title: Morse Account / Actor Claiming
sidebar_position: 2
---

AFTER the canonical `MorseAccountState` has been imported onchain by the authority (see: [State Export / Transform / Import](./morse-migration.md#state-export--transform--validate)), Morse account/stake-holders can "claim" their Morse accounts onchain on Shannon.

### Onchain Actors & Messages

Morse account holders who are staked as either applications or suppliers (aka "servicers") on Morse, can claim their accounts **as a staked actor** on Shannon; maintaining their existing actor stake.
Depending on whether the account is staked, and as which actor type, the corresponding claim message MUST be used.

:::important
I.e.: An unstaked account CANNOT claim as a staked actor, and staked accounts MUST claim as their actor type.
:::

:::note
Account balances and stakes MAY be adjusted prior to "Judgement Day" OR after claiming on Shannon.
:::

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

:::warning TODO_IN_THIS_COMMIT: add structure and copy...

- CLI use cases and examples
    - account
    - application
    - supplier
:::

```bash
$ poktrolld migrate claim-account ./pocket-account-8b257c7f4e884e49bafc540d874f33f91436e1dc.json --from app1
Enter Decrypt Passphrase: 
MsgClaimMorseAccount {
  "shannon_dest_address": "pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4",
  "morse_src_address": "8B257C7F4E884E49BAFC540D874F33F91436E1DC",
  "morse_signature": "hLGhLRjP6jgP6wgOIaYFxIxT3z4jb4IBDKovMkX5AqUsOqdF+rEIO5aofOKnmYW9BkqL0v2DfUfE3nj25FNhBA=="
}
Confirm MsgClaimMorseAccount: y/[n]: 
```
