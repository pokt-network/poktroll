---
title: Claim & Proof Lifecycle
sidebar_position: 2
---

# Claim & Proof Lifecycle <!-- omit in toc -->

:::warning

This part of the documentation is just an INITIAL DRAFT and requires deep
understanding of the Pocket Network protocol. It is currently aiming to just
be a reference and not provide a coherent narrative that is easily accessible
to all readers.

TODO(@Olshansk): Iterate on this doc & link to governance params.

TODO(@red-0ne): Review this document and submit a PR with updates & edits.

:::

- [Introduction](#introduction)
- [Session](#session)
  - [Session Duration](#session-duration)
  - [Session End](#session-end)
- [Claim](#claim)
  - [Claim Protobuf](#claim-protobuf)
  - [CreateClaim Transaction](#createclaim-transaction)
  - [CreateClaim Validation](#createclaim-validation)
  - [Claim Window](#claim-window)
- [Proof](#proof)
  - [Proof Protobuf](#proof-protobuf)
  - [SubmitProof Transaction](#submitproof-transaction)
  - [SubmitProof Validation](#submitproof-validation)
  - [Proof Window](#proof-window)
- [Proof Security](#proof-security)
  - [Merkle Leaf Validation](#merkle-leaf-validation)
  - [Merkle Proof Selection](#merkle-proof-selection)
    - [Example: Example Sparse Merkle Sum Trie (SMST)](#example-example-sparse-merkle-sum-trie-smst)
    - [Example 1: Path to leaf at full depth](#example-1-path-to-leaf-at-full-depth)
    - [Example 2: Path to leaf at partial depth](#example-2-path-to-leaf-at-partial-depth)
    - [Example 3: Path to empty node](#example-3-path-to-empty-node)
- [Full Lifecycle](#full-lifecycle)

## Introduction

The `Claim & Proof` lifecycle is a fundamental part of the Pocket Network protocol.

At a high-level, it is an adaptation of a well-known `commit & reveal` paradigm used
in various blockchain application such as [ENS](https://docs.ens.domains/contract-api-reference/.eth-permanent-registrar/controller).

:::note

For the purpose of explaining the `Claim & Proof` lifecycle, we will remove
`Gateways` from the discussion and assume that `Applications` are directly
interacting with the `Suppliers`.

:::

```mermaid
---
title: Claim & Proof Lifecycle
---
sequenceDiagram
    actor A as Application(s)
    actor S as Supplier(s)
    participant PN as Pocket Network<br>(Distributed Ledger)

    loop Session Duration
        note over A,S: off-chain
        A ->> +S: Relay Request
        S ->> S: Insert Leaf into <br> Sparse Merkle Sum Trie
        S ->> -A: Relay Response
    end

    par For every (App, Supplier, Service)
        note over S, PN: Claim Window (Wait a few blocks)
        S ->> PN: CreateClaim(Session, MerkleRootHash)
        PN -->> S: Seed for merkle branch
        note over S, PN: Proof Window (Wait a few blocks)
        S ->> PN: SubmitProof(Session, ClosestMerkleProof)
        PN ->> PN: Validate Proof & <br> Validate Leaf
        PN -->> S: Increase account balance (emission)
        PN -->> A: Deduct staked balance (burn)
    end
```

## Session

A session is a necessary pre-requisite for the `Claim & Proof` lifecycle to work.
See [Session](./session.md) for more details.

### Session Duration

After a session is initiated, the majority of it is handled `off-chain`,
as `Applications` make RPC requests (`relays`) to the `Supplier`.

### Session End

After a session ends, the Claim & Proof Lifecycle can be decomposed, at a high-level,
into the following steps.

```mermaid
timeline
    title Post Session Proof Validation
    Session Ends <br> (Protocol)
        : Recompute SMST root & sum (compute units)
        : Flush and store SMST to local disk
    CreateClaim <br> (Supplier)
        : Wait for Claim Window to open
        : Submit CreateClaim Transaction <br>(root, sum, session, app, supplier, service, etc...)
        : Claim stored on-chain
    SubmitProof <br> (Supplier)
        : Wait for Proof Window to open
        : Retrieve seed (entropy) from on-chain data (block hash)
        : Generate Merkle Proof for path in SMST based on seed
        : Submit SubmitProof Transaction <br>(session, merkle proof, leaf, etc...)
        : Proof stored on-chain
    Proof Validation <br> (Protocol)
        : Retrieve on-chain Claims that need to be settled
        : Retrieve corresponding on-chain Proof for every Claim
        : Validate leaf difficulty
        : Validate Merkle Proof
        : Validate Leaf Signature
        : Burn Application Stake proportional to sum
        : Inflate Supplier Balance  proportional to sum
```

## Claim

A `Claim` is a structure submitted on-chain by a `Supplier` claiming to have done
some amount of work in servicing `relays` for `Application`.

Exactly one claim exists for every `(Application, Supplier, Session)`.

A `Claim` forces a `Supplier` to commit to have done `sum` work during a `Session` for
a certain `Application`. The `sum` in the root of the SMST is the amount of work
done. Each leaf has a different `weight` depending on the number of _"compute units"_
that were necessary to service that request.

_TODO_DOCUMENT(@Olshansk): Link to a document on compute units once it it written._

### Claim Protobuf

A serialized version of the `Claim` is stored on-chain.

You can find the definition for the [Claim structure here](../../../proto/poktroll/proof/claim.proto).

### CreateClaim Transaction

A `CreateClaim` transaction can be submitted by a `Supplier` to store a claim `on-chain`.

You can find the definition for the [CreateClaim Transaction here](../../../proto/poktroll/proof/tx.proto).

### CreateClaim Validation

_TODO(@bryanchriswhite, @Olshansk): Update this section once [msg_server_create_claim.go](./../../../x/proof/keeper/msg_server_create_claim.go) is fully implemented._

### Claim Window

After a `Session` ends, a `Supplier` has several blocks, a `Claim Window`, to submit
a `CreateClaim` transaction containing a `Claim`. If it is submitted too early
or too late, it will be rejected by the protocol.

If a `Supplier` fails to submit a `Claim` during the Claim Window, it will forfeit
any potential rewards it could earn in exchange for the work done.

_TODO(@Olshansk): Link to the governance params governing this once implemented._

## Proof

A `Proof` is a structure submitted on-chain by a `Supplier` containing a Merkle
Proof to a single pseudo-randomly selected leaf from the corresponding `Claim`.

At most one `Proof` exists for every `Claim`.

A `Proof` is necessary for the `Claim` to be validated so the `Supplier` can be
rewarded for the work done.

_TODO_DOCUMENT(@Olshansk): Link to a document on compute units once it it written._

### Proof Protobuf

A serialized version of the `Proof` is stored on-chain.

You can find the definition for the [Proof structure here](../../../proto/poktroll/proof/proof.proto)

### SubmitProof Transaction

A `SubmitProof` transaction can be submitted by a `Supplier` to store a proof `on-chain`.

If the `Proof` is invalid, or if there is no corresponding `Claim` for the `Proof`, the
transaction will be rejected.

You can find the definition for the [SubmitProof Transaction here](../../../proto/poktroll/supplier/tx.proto).

### SubmitProof Validation

_TODO(@bryanchriswhite, @Olshansk): Update this section once [msg_server_submit_proof.go](./../../../x/proof/keeper/msg_server_submit_proof.go) is fully implemented._

### Proof Window

After the `Proof Window` opens, a `Supplier` has several blocks, a `Proof Window`,
to submit a `SubmitProof` transaction containing a `Proof`. If it is submitted too
early or too late, it will be rejected by the protocol.

If a `Supplier` fails to submit a `Proof` during the Proof Window, the Claim will
expire and it it will forfeit any previously claimed work done.

_TODO(@Olshansk): Link to the governance params governing this once implemented._

## Proof Security

In addition to basic validation as part of processing `SubmitProof` to determine
whether or not the `Proof` should be stored on-chain, there are several additional
deep cryptographic validations needed:

1. `Merkle Leaf Validation`: Proof of the off-chain `Supplier`/`Application` interaction during the Relay request & response.
2. `Merkle Proof Selection`: Proof of the amount of work done by the `Supplier` during the `Session`.

:::note

TODO: Link to tokenomics and data integrity checks for discussion once they are written.

:::

### Merkle Leaf Validation

The key components of every leaf in the `Sparse Merkle Sum Trie` are shown below.

After the leaf is validated, two things happen:

1. The stake of `Application` signing the `Relay Request` is decreased through burn
2. The account balance of the `Supplier` signing the `Relay Response` is increased through mint

The validation on these signatures is done on-chain as part of `Proof Validation`.

```mermaid
graph LR
    subgraph Sparse Merkle Sum Trie Leaf
        subgraph Metadata
            S["Session"]
            W["Weight (compute units)"]
        end
        subgraph Signed Relay Request
            direction TB
            Req[Relay Request Data]
            AppSig(ApplicationSignature)
            AppSig o-.-o Req
        end

        subgraph Signed Relay Response
            direction TB
            Res[Relay Response Data]
            SupSig(SupplierSignature)
            SupSig o-.-o Res
        end
    end
```

### Merkle Proof Selection

Before the leaf itself is validated, we need to make sure if there is a valid
Merkle Proof for the associated pseudo-random path computed on-chain.

Since the path that needs to be proven uses an on-chain seed after the `Claim`
has been submitted, it is impossible to know the path in advance.

Assume a collision resistant hash function `H` that takes a the `block header hash`
as the `seed` and maps it to a `path` in the `Merkle Trie` key space.

#### Example: Example Sparse Merkle Sum Trie (SMST)

Below is an example of a `Sparse Merkle Sum Trie` where the paths can be at
most `5` bits (for example purposes).

:::note

Extension nodes are ommitted and shown via `0bxxxxx` as part of the tree edges

:::

Legend:

- 🟥 - Root node
- 🟦 - Inner node
- 🟩 - Leaf node
- 🟫 - Empty Node
- 🟨 - Included in Merkle Proof
- ⬚🟨 - Computed as Part of Merkle Proof
- ⬛ - Not used in the diagram node

```mermaid
graph TB
    classDef redNode fill:#ff0000, color:#ffffff;
    classDef greenNode fill:#00b300, color:#ffffff;
    classDef blueNode fill:#0000ff, color:#ffffff;
    classDef yellowNode fill:#fff500, color:#ffa500
    classDef brownNode fill:#964B00, color:#ffffff;

    %% Define root node
    R[sum=9<br>root]

    %% Height = 1
    R -- 0 --> N1[sum=5<br>0b0]
    R -- 1 --> N2[sum=4<br>0b1]

    %% Height = 2
    N1 -- 0 --> E1[sum=0<br>0b00xxx]
    N1 -- 1 --> N3[sum=5<br>0b01]
    N2 -- 0b10xxx --> L1[weight=1<br>0b10000]
    N2 -- 1 --> N4[sum=3<br>0b11]

    %% Height = 3
    N3 -- 0b010xx --> L2[weight=2<br>0b01000]
    N3 -- 0b011xx --> L3[weight=3<br>0b01100]
    N4 -- 0 --> E2[sum=0<br>0b100xx]
    N4 -- 1 --> N5[sum=3<br>0b111]

    %% Height = 4
    N5 -- 0b1110x --> L4[weight=1<br>0b11100]
    N5 -- 1 --> N6[sum=2<br>0b1111]

    %% Height = 5
    N6 -- 0 --> L5[weight=1<br>0b11110]
    N6 -- 1 --> L6[weight=1<br>0b11111]

    class R redNode;
    class L1,L2,L3,L4,L5,L6 greenNode;
    class N1,N2,N3,N4,N5,N6 blueNode;
    class E1,E2 brownNode;
```

#### Example 1: Path to leaf at full depth

```mermaid
---
title: Path to leaf at full depth (path=0b11111)
---
graph TB
    %% Define a class for red nodes
    classDef redNode fill:#ff0000, color:#ffffff;
    classDef greenNode fill:#00ff00, color:#ffffff;
    classDef blueNode fill:#0000ff, color:#ffffff;
    classDef yellowNode fill:#fff500, color:#ffa500
    classDef yellowBorderNode stroke:#fff500, stroke-width:4px, stroke-dasharray: 5 5

    %% Define root node
    R[sum=9<br>root]

    %% Height = 1
    R -- 0 --> N1[sum=5<br>0b0]
    R -- 1 --> N2[sum=4<br>0b1]

    %% Height = 2
    N1 -- 0 --> E1[sum=0<br>0b00xxx]
    N1 -- 1 --> N3[sum=5<br>0b01]
    N2 -- 0b10xxx --> L1[weight=1<br>0b10000]
    N2 -- 1 --> N4[sum=3<br>0b11]

    %% Height = 3
    N3 -- 0b010xx --> L2[weight=2<br>0b01000]
    N3 -- 0b011xx --> L3[weight=3<br>0b01100]
    N4 -- 0 --> E2[sum=0<br>0b100xx]
    N4 -- 1 --> N5[sum=3<br>0b111]

    %% Height = 4
    N5 -- 0b1110x --> L4[weight=1<br>0b11100]
    N5 -- 1 --> N6[sum=2<br>0b1111]

    %% Height = 5
    N6 -- 0 --> L5[weight=1<br>0b11110]
    N6 -- 1 --> L6[weight=1<br>0b11111]

    class R redNode;
    class L1,L4,L5,E2,N1 yellowNode;
    class N6,N5,N4,N2 yellowBorderNode;
    class L6 greenNode;
```

#### Example 2: Path to leaf at partial depth

```mermaid
---
title: Path to leaf at partial depth (path=0b01100)
---
graph TB
    %% Define a class for red nodes
    classDef redNode fill:#ff0000, color:#ffffff;
    classDef greenNode fill:#00ff00, color:#ffffff;
    classDef blueNode fill:#0000ff, color:#ffffff;
    classDef yellowNode fill:#fff500, color:#ffa500
    classDef yellowBorderNode stroke:#fff500, stroke-width:4px, stroke-dasharray: 5 5

    %% Define root node
    R[sum=9<br>root]

    %% Height = 1
    R -- 0 --> N1[sum=5<br>0b0]
    R -- 1 --> N2[sum=4<br>0b1]

    %% Height = 2
    N1 -- 0 --> E1[sum=0<br>0b00xxx]
    N1 -- 1 --> N3[sum=5<br>0b01]
    N2 -- 0b10xxx --> L1[weight=1<br>0b10000]
    N2 -- 1 --> N4[sum=3<br>0b11]

    %% Height = 3
    N3 -- 0b010xx --> L2[weight=2<br>0b01000]
    N3 -- 0b011xx --> L3[weight=3<br>0b01100]
    N4 -- 0 --> E2[sum=0<br>0b100xx]
    N4 -- 1 --> N5[sum=3<br>0b111]

    %% Height = 4
    N5 -- 0b1110x --> L4[weight=1<br>0b11100]
    N5 -- 1 --> N6[sum=2<br>0b1111]

    %% Height = 5
    N6 -- 0 --> L5[weight=1<br>0b11110]
    N6 -- 1 --> L6[weight=1<br>0b11111]

    class R redNode;
    class E1,N2,L2 yellowNode;
    class N1,N3 yellowBorderNode;
    class L3 greenNode;
```

#### Example 3: Path to empty node

```mermaid
---
title: Path to leaf at partial depth (path=0b100xx->0b10000)
---
graph TB
    classDef redNode fill:#ff0000, color:#ffffff;
    classDef greenNode fill:#00ff00, color:#ffffff;
    classDef greenNodeDark fill:#067620, color:#ffffff;
    classDef blueNode fill:#0000ff, color:#ffffff;
    classDef yellowNode fill:#fff500, color:#ffa500
    classDef yellowBorderNode stroke:#fff500, stroke-width:4px, stroke-dasharray: 5 5

    %% Define root node
    R[sum=9<br>root]

    %% Height = 1
    R -- 0 --> N1[sum=5<br>0b0]
    R -- 1 --> N2[sum=4<br>0b1]

    %% Height = 2
    N1 -- 0 --> E1[sum=0<br>0b00xxx]
    N1 -- 1 --> N3[sum=5<br>0b01]
    N2 -- 0b10xxx --> L1[weight=1<br>0b10000]
    N2 -- 1 --> N4[sum=3<br>0b11]

    %% Height = 3
    N3 -- 0b010xx --> L2[weight=2<br>0b01000]
    N3 -- 0b011xx --> L3[weight=3<br>0b01100]
    N4 -- 0 --> E2[sum=0<br>0b100xx]
    N4 -- 1 --> N5[sum=3<br>0b111]

    %% Height = 4
    N5 -- 0b1110x --> L4[weight=1<br>0b11100]
    N5 -- 1 --> N6[sum=2<br>0b1111]

    %% Height = 5
    N6 -- 0 --> L5[weight=1<br>0b11110]
    N6 -- 1 --> L6[weight=1<br>0b11111]

    class R redNode;
    class N1,N5 yellowNode;
    class N4,N2 yellowBorderNode;
    class E2,L1 greenNode;
    class E2 greenNodeDark;
```

## Full Lifecycle

The following diagram was taken from the [Relay Mining whitepaper](https://arxiv.org/pdf/2305.10672.pdf),
and is an alternative view of the full lifecycle described above.
It is here for reference purposes.

```mermaid
sequenceDiagram
    actor A as Application
    actor S as Servicer 1..N
    actor Svc as Service / Data Node
    participant W as World State
    alt Step 2. Start Session: Blocks [B, B+W)
        A ->> +W: GetSessionData(AppPubKey, Svc, BlockHeight, ...)
        S ->> W: GetSessionData(AppPubKey, Svc, BlockHeight, ...)
        W ->> S: Session(Header, [Servicer])
        W ->> -A: Session(Header, [Servicer])
    end

    loop Step 3. During Session: Blocks [B, B+W)
        A ->> +S: Signed(Relay)
        S ->> S: Relay Validation
        S ->> +Svc: Service(Request)
        Svc ->> -S: Response
        S ->> S: 1. Compute hash<br>2. Insert in SMT<br>3. Decrement token count
        S ->> -A: Signed(Response)
    end

    alt Step 4. After Session: Blocks [B+W+1, B+W+1+L)
        S ->> W: Claim(SMT Root Commitment)
        note over S,W: Wait L blocks
        S ->> +W: GetProofRequest(SessionHeader, ServicerPubKey, AppPubKey, ...)
        W ->> -S: ProofRequestDetails
        S ->> +W: Proof(SMT Branch Reveal)
        W ->> S: Token Rewards (Increase Servicer Balance)
        W ->> -A: Token Burn (Decrease Application Stake)
    end
```
