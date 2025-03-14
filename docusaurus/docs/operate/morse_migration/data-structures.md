---
title: Data Structures
sidebar_position: 4
---

## Table of Contents <!-- omit in toc -->

- [Offchain Shannon Structure(s)](#offchain-shannon-structures)
- [Onchain Shannon Structure(s)](#onchain-shannon-structures)

## Offchain Shannon Structure(s)

```mermaid
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

## Onchain Shannon Structure(s)

```mermaid
classDiagram
    
class MorseAccountState {
    accounts: []MorseClaimableAccount
}    
        
class MorseClaimableAccount {
    shannon_dest_address: string
    morse_src_address: string
    public_key: bytes
    unstaked_balance: cosmos.base.v1beta1.Coin
    supplier_stake: cosmos.base.v1beta1.Coin
    application_stake: cosmos.base.v1beta1.Coin
    claimed_at_height: int64
}

MorseAccountState --* MorseClaimableAccount

class MsgCreateMorseAccountState {
    authority: string
    morse_account_state: MorseAccountState
    morse_account_state_hash: bytes
}
MsgCreateMorseAccountState --* MorseAccountState

class MsgClaimMorseAccount {
    shannon_dest_address: string
    morse_src_address: string
    morse_signature: bytes
}
MsgClaimMorseAccount ..> MorseClaimableAccount: morse_src_address ref.

class MsgClaimMorseApplication {
    shannon_dest_address: string
    morse_src_address: string
    morse_signature: bytes
    service_config: shared.ApplicationServiceConfig
}
MsgClaimMorseApplication ..> MorseClaimableAccount: morse_src_address ref.

class MsgClaimMorseSupplier {
    shannon_owner_address: string
    shannon_operator_address: string
    morse_src_address: string
    morse_signature: bytes
    services: []shared.SupplierServiceConfig
}
MsgClaimMorseSupplier ..> MorseClaimableAccount: morse_src_address ref.
```

For more info regarding onchain message usage, see [onchain actors & messages](./claiming.md#onchain-actors--messages).
