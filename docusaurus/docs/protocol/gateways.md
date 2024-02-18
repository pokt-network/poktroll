---
title: Gateways
sidebar_position: 4
---

# Gateways <!-- omit in toc -->

:::warning

This part of the documentation is just an initial draft and requires deep
understanding of the Pocket Network protocol. It is currently aiming to just
be a reference and not provide a coherent narrative that is easily accessible
to all readers.

TODO(@Olshansk): Iterate on this doc & link to governance params.

:::

The [Gateway Actor](./../actors/gateway.md) section covers what a Gateway is.
Recall that it is a permissionless protocol actor to whom the Application can
**optionally** delegate on-chain trust in order to perform off-chain operations.

This section aims to cover the cryptographic aspects of Gateway interactions,
trust delegation, and how they fit into the Pocket Network protocol.

## Modes of Operation

There are three modes of operation to interact with the Suppliers on the network:

1. **Sovereign Application**
2. **Delegating Application**
3. **Gateway Application**

For the purposes of this discussion, it is important to note that an `Application`
and `Gateway` are on-chain actors/records that stake POKT to participate in the
network. The term `Client` is used to represent an application running on a user's
device, such as a smartphone or a web browser.

The goal of Gateways is to enable free-market off-chain economics tie into
on-chain interactions.

### Sovereign Application

A Sovereign Application is one where the `Client` manages its own on-chain `Application`
and interacts with the Pocket Supplier Network directly.

The Application is responsible for:

- Protecting it's own `Application` private key on the `Client`
- Maintaining and updating it's own on-chain stake to pay for `Supplier` services
- Determining which `Supplier` to use from the available list in the session

```mermaid
sequenceDiagram
    actor A as Application <br> (Client)
    participant DA as Pocket DA Layer
    actor S as Supplier(s)

    A ->> +DA: StartSession(App, Block, ...)
    DA ->> -A: SessionData([Suppliers], ...)

    loop Session Duration
        A ->> A: Sign Relay Request
        A ->>+ S: Signed Relay Request
        S ->> S: Validate App Signature & <br>App Session Limits
        alt App exceeds session limits
            S ->> A: Reject Request
        else App within session limits
            S ->> S: Handle Request &<br>Sign Response
            S ->>- A: Signed Relay Response
        end
    end

    S -->> DA: Claim & Proof Lifecycle
```

### Delegating Application

A Delegated Application is one where an `Application` delegates to one or more
`Gateways`. Agreements (authentication, payments, etc) between the `Client` and
`Gateway` are then managed off-chain, but payment for the on-chain `Supplier`
services still comes from the `Application`s stake.

The Application is responsible for:

- Protecting it's own `Application` private key somewhere in hot/cold storage
- Maintaining and updating it's own on-chain stake to pay for `Supplier` services
- Managing, through (un)delegation, which Gateway(s) can sign requests on ts behalf

The Gateway is responsible for:

- Providing tooling and infrastructure to coordinate with the `Client`
- Determining which `Supplier` to use from the available list in the session

```mermaid
sequenceDiagram
    actor A as Application
    participant C as Client
    actor G as Gateway(s)
    participant DA as Pocket DA Layer
    actor S as Supplier(s)

    A -->> +DA: Delegate([Gateway])

    G ->> +DA: StartSession(App, Block, ...)
    DA ->> -G: SessionData([Suppliers], ...)

    note over C,G: Client-Gateway Handshake <br> (e.g. OAuth, etc...)

    loop Session Duration
        C ->> G: Request
        G ->> G: Sign Relay Request
        G ->>+ S: Signed Relay Request
        S ->> S: Validate Ring(App/Gateway) Signature & <br> App Session Limits
        alt App exceeds session limits
            S ->> G: Reject Request
            G ->> C: Rejected Response <br>(or backup response)
        else App within session limits
            S ->> S: Handle Request &<br>Sign Response
            S ->> -G: Signed Relay Response
            G ->> C: Response
        end
    end

    S -->> DA: Claim & Proof Lifecycle
```

### Gateway Application

A Gateway Application is one where the `Gateway` takes full onus, on behalf of
`Client`s to manage all on-chain `Application` interactions to access the
Pocket `Supplier` Network. Agreements (authentication, payments, etc) between
the `Client` and `Gateway` are then managed off-chain, and payment for the
on-chain `Supplier` services will comes from the `Application`s stake, which
is now maintained by the `Gateway`.

It is responsible for:

The Gateway is responsible for:

- Protecting it's own `Application` private key somewhere in hot/cold storage
- Maintaining and updating it's own on-chain stake to pay for `Supplier` services
- Providing tooling and infrastructure to coordinate with the `Client`
- Determining which `Supplier` to use from the available list in the session

```mermaid
sequenceDiagram
    participant C as Client
    actor G as Gateway(s) <br> (Application(s))
    participant DA as Pocket DA Layer
    actor S as Supplier(s)

    G ->> +DA: StartSession(App, Block, ...)
    DA ->> -G: SessionData([Suppliers], ...)

    note over C,G: Client-Gateway Handshake <br> (e.g. OAuth, etc...)

    loop Session Duration
        C ->> G: Request
        G ->> G: Sign Relay Request
        G ->>+ S: Signed Relay Request
        S ->> S: Validate Ring(App/Gateway) Signature & <br> App Session Limits
        alt App exceeds session limits
            S ->> G: Reject Request
            G ->> C: Rejected Response <br>(or backup response)
        else App within session limits
            S ->> S: Handle Request &<br>Sign Response
            S ->> -G: Signed Relay Response
            G ->> C: Response
        end
    end

    S -->> DA: Claim & Proof Lifecycle
```

## Application -> Gateway Delegation

An Application that chooses to delegate trust to a gateway by submitting a
one-time `DelegateMsg` transaction. Once this is done, the `Gateway` will be
able to sign relay requests on behalf of the `Application` that'll use the
`Application`s on-chain stake to pay for service to access the Pocket `Supplier` Network.

This can be done any number of times, so an `Application` can delegate to multiple
`Gateways` simultaneously.

```mermaid
---
title: Application -> Gateway (un)Delegation
---
sequenceDiagram
    actor A as Application
    participant DA as Pocket DA Layer
    actor G as Gateway
    A ->> A: Prepare & Sign <br> Delegation Transaction
    A ->>+ DA: Delegate(GatewayPubKey)
    DA ->>- A: ok
    note over A,G: Gateway can now sign <br>relay requests on behalf of Application
    A ->> A: Prepare & Sign <br> Undelegation Transaction
    A ->>+ DA: Undelegate(GatewayPubKey)
    DA ->>- A: ok
    note over A,G: Gateway can now longer sign <br>relay requests on behalf of Application
```

### Ring Signature Verification

[Ring Signatures](https://en.wikipedia.org/wiki/Ring_signature) will be used in order to allow both the Application and the Gateway to sign the Relay.

```mermaid
flowchart
    S[Supplier]

    subgraph SA[Sovereign Application]
        subgraph SARing[Ring Signature]
            A1[Application 1]
            A1 <--> A1
        end
    end

    subgraph DA[Delegating Application]
        subgraph DARing
            A2[Application 2]
            G1[Gateway 1]
            G2[Gateway 2]
            A2 <--> G1
            G1 <--> G2
            G2 <--> A2
        end
    end

    subgraph AG[Application Gateway]
        subgraph AGRing
            G3["Gateway 3<br>(Application 3)"]
            G3 <--> G3
        end
    end


    AG --Signature--> S
    DA --Signature--> S
    SA --Signature--> S

    S-->|Validate Signature| S
```

```mermaid
---
title: Signature Validation for Delegating Application
---
stateDiagram-v2
    state "Get Gateways the App<br>delegated to: [P1, P2]" as getGateways
    state "Is Relay Request signed by one of:<br>[Application 2, Gateway1, Gateway2]?" as sigCheck

    state "Valid (should service relay)" as Valid
    state "Invalid (do not service relay)" as Invalid

    [*] --> getGateways
    getGateways --> sigCheck

    sigCheck --> Valid: Yes
    sigCheck --> Invalid: No
```

## Gateway Off-Chain Operations

- altruist
- Check
- Client Side Challenge & Response
- Proof w/ that
- Etc.
- Session dispatching
- Pocket Network buisness logic
- Supplier selection and QoS management
-
