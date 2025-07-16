# Pocket IBC Interoperability

## 📖 Table of Contents

- [IBC Primer (Core Concepts)](#1-ibc-primer-core-concepts)
    - [IBC Protocol Layers](#11-ibc-protocol-layers)
    - [Key Components](#12-key-components)
- [IBC Transfer & ICS-20](#2-ibc-transfer--ics20)
    - [Pain Points in Vanilla ICS-20](#21-%EF%B8%8F-pain-points-in-vanilla-ics20)
- [Interchain Accounts (ICA / ICS-27)](#3-interchain-accounts-ica--ics27)
    - [Host & Controller Roles](#31-host--controller-roles)
        - [Host Chain](#311-host-chain)
        - [Controller Chain](#312-controller-chain)
    - [ICA Queries](#32-ica-queries)
    - [Conditional Clients](#33-conditional-clients)
    - [ICA Features Summary](#34--ica-features-summary)
- [IBC Middleware & Apps](#4-ibc-middleware--apps)



## 1. IBC Primer (Core Concepts)

The **Inter‑Blockchain Communication Protocol (IBC)** is a standardized, modular framework enabling secure, permissionless communication between heterogeneous blockchains.

### 1.1 IBC Protocol Layers

* **IBC/TAO (Transport, Authentication, Ordering)** – the cross‑chain infrastructure handling packet lifecycle.
* **IBC/App** – the application layer, including modules like ICS‑20 (fungible token transfers), ICS‑721 (NFTs), ICS‑27 (Interchain Accounts), and others.

### 1.2 Key Components:

1. [**Client**](https://tutorials.cosmos.network/academy/3-ibc/4-clients.html) – each chain maintains a light client of the other.
2. [**Connection**](https://tutorials.cosmos.network/academy/3-ibc/2-connections.html) – four‑step handshake (`ConnOpenInit/Try/Ack/Confirm`).
3. [**Channel**](https://tutorials.cosmos.network/academy/3-ibc/3-channels.html) – attached to a connection, providing ordered or unordered packet transport.
4. [**Relayer**](https://tutorials.cosmos.network/academy/2-cosmos-concepts/13-relayer-intro.html) – off‑chain service that listens for IBC events and relays packet proof messages between chains (e.g. [Hermes](https://hermes.informal.systems/)).


## 2. IBC Transfer & ICS‑20

IBC enables **cross-chain fungible token transfers** via the **ICS‑20** standard.

:::tip Querying Localnet IBC State

See [Localnet IBC Environment -> Testing Localnet IBC -> Transfers](./localnet.md#transfers) for examples of localnet IBC transfers.

:::

### 2.1 ⚠️ Pain Points in Vanilla ICS‑20

_See [IBC Middleware & Apps -> Packet Forward Middleware (PFM)](#4-ibc-middleware--apps) for a comparison._

* **Denom Tracing Creates Fragmented Balances**
  Each transfer hops add a new layer to the denom (e.g., `transfer/channel-1/uatom`) hashed to `ibc/...`. If a user sends tokens across multiple hops and then back on a different path, they end up with separate voucher balances—even if from the same source token ([tutorials.cosmos.network](https://tutorials.cosmos.network/tutorials/6-ibc-dev/), [strange.love](https://strange.love/blog/introducing-packet-forward-middleware)).

* **Manual Multi-Hop Transfers**
  Users must execute each hop one at a time—signing multiple transactions and manually unwinding paths to return tokens to their native form.


## 3 Interchain Accounts (ICA / ICS‑27)

### 3.1 Host & Controller Roles

#### 3.1.1 **Host Chain**

* Listens on fixed port **`icahost`** and executes transactions received via IBC.
* Handles channel closure confirmation (via `ChanCloseConfirm`), but does **not** initiate closures.
* Implements transaction execution logic remotely via IBC packets.
  *(See the [ICS‑27 spec](https://github.com/cosmos/ibc/blob/master/spec/app/ics-027-interchain-accounts/README.md) for more detail)*

#### 3.1.1 **Controller Chain**

* Uses dynamic ports prefixed **`icacontroller-<owner-address>`**.
* Exposes three key operations:

    * **`MsgRegisterInterchainAccount`** – Establishes ICA channel and host-side account.
    * **`MsgSendTx`** – Sends `EXECUTE_TX` IBC packets for transaction execution.
    * **`MsgModuleQuerySafe`** – Performs safe ICA Queries before execution.
* These messages are part of the ICA controller gRPC API. 

### 3.2 ICA Queries

* Enables **`MsgModuleQuerySafe`**, allowing a controller to query host-side modules marked `module_query_safe` within a single transaction, returning the response in the IBC acknowledgment. _(Added in **ibc-go v7.5.0**)_
* Typical use cases include checking account balances, validator status, or token metadata before executing actions—avoiding unnecessary failures. 

### 3.3 Conditional Clients

* **Conditional Clients** allow one IBC light client to condition its state verification on another client's state (via `VerifyMembership`). _(Added in **ibc-go v8.3.0**)
* Vital for modular or rollup-based systems where data inclusion (e.g., in a DA layer) must be confirmed before verifying other packet commitments.
* Supports both **Go-native** and **WASM**-based clients.

### 3.4 🧭 ICA Features Summary

| Feature                 | Role              | Introduced | Description                                                        |
| ----------------------- | ----------------- | ---------- | ------------------------------------------------------------------ |
| **Host Chain**          | Receiver          | vX+        | Fixed port `icahost`, executes packets via ICA                     |
| **Controller Chain**    | Sender            | v3+        | Dynamic port; enables `Register`, `SendTx`, `ModuleQuerySafe`      |
| **ICA Queries**         | Controller → Host | v7.5.0     | Adds `MsgModuleQuerySafe` for read-before-write flows              |
| **Conditional Clients** | Light Clients     | v8.3.0     | Enables inter-client proof dependencies for modular chains/rollups |


## 4. IBC Middleware & Apps

### 4.1 Packet Forward Middleware (PFM)

[**Packet forward middleware**](https://github.com/cosmos/ibc-apps/tree/modules/rate-limiting/v8.1.0/middleware/packet-forward-middleware) is an optional [**IBC middleware**](https://github.com/cosmos/ibc-apps/tree/modules/rate-limiting/v8.1.0/middleware) (PFM / ICS-30) that enhances vanilla ICS-20 by offering:

- **Atomic multi-hop transfers in one transaction**: users issue a single `msgtransfer` with a memo containing a json `forward` route. pfm handles a → b → c … hops in one go — no multiple signatures required.
- **Single final acknowledgment**: origin chain receives only one ack after **all** hops succeed or fail. intermediate responses and retries are handled internally.
- **Automatic denom path unwinding**: ensures fungibility by normalizing paths upon token return—addresses fragmented denominations.
- **Retries, timeouts & fee options**: intermediate chains can auto-retry hops, trigger refunds, or deduct forwarding fees.

**🔄 Comparison: Vanilla ICS‑20 vs. PFM**

| Feature                   | Vanilla ICS‑20            | With PFM                                               |
|---------------------------|---------------------------|--------------------------------------------------------|
| Multi-hop transfers       | Manual, hop-by-hop        | Auto, single-transaction flow                         |
| Acknowledgements          | Per-hop                   | One final ack at origin                                |
| Denom fragmentation       | Yes (voucher path issues) | No—automated unwinding preserves fungibility           |
| Fee & developer support   | None                      | Optional fee and developer-configurable routing options |

PFM significantly improves ux by abstracting ibc’s routing complexity and minimizing fragmented balances.  
Highly recommended for apps needing **cross-chain composition** or **simplified token routing**.

```
TODO(@bbryanchriswhite, #1568): document/link to how to use pfm.
```

