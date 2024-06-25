---
title: Relay Mining
sidebar_position: 3
---

# Relay Mining <!-- omit in toc -->

:::warning

TODO_DOCUMENT(@Olshansk): This is just a placeholder. Use the [relay mining presentation](https://docs.google.com/presentation/d/1xlCGzS_oHXJOzvcu-jHZUfmhD3qeVCzc6SUSJijTuJ4/edit#slide=id.p) and
the [relay mining paper](https://arxiv.org/abs/2305.10672) as a reference for writing this.

:::

- [Introduction](#introduction)

## Introduction

tl;dr Modulate on-chain difficulty up (similar to Bitcoin) so we can accommodate
surges in relays and have no upper limit on the number of relays per session.

Relay Mining is the only solution in Web3 to incentivize read-only requests
and solve for the problem of high volume: `how can we scale to billions or trillions
of relays per session`.

This complements the design of [Probabilistic Proofs](./probabilistic_proofs.md)
to solve for all scenarios.

## Relay Sessions

[Sessions](./session) effectively group relays into time-wise batches. During each session,
`Application`s and/or `Gateway`s can submit relays to one or more of the `Supplier`s
in the current session for servicing.

### Sovereign Application

An `Application` can act as a "soverign application" (i.e. its own `Gateway`).
In this case, the ring used to sign relay requests is constructed only from
the `Application`'s public key.

```mermaid
---
title: Sovereign Application - RPC Request/Response & Claim/Proof
---
sequenceDiagram

actor user as User
participant app as Gateway (Sovereign Application)
participant pokt as Pokt Network
participant sup as Supplier
participant rpc as RPC Server

app-)pokt: Stake for service(s)
sup-)pokt: Stake for service(s)

loop Session N

loop Every relay request

Note over user,rpc: Ref: RPC Request/Response Relay (Sovereign Application)

end
end


break Wait for session N grace period to end
    sup->>sup: Persist unclaimed & unproven session trees
end

loop Session N+GracePeriod+1

loop Every application session from session N

Note over pokt,sup: ref: Supplier Claim/Proof

end

end
```
> **See**: [Legend > Sequence Diagram](#sequence-diagram)

### Delegated Application

An `Application` can also be delegated to one or more `Gateway`s. In this case
the ring used to sign relay requests is constructed from the `Application`'s
public key and the public keys of the all `Gateway`s it is delegated to at the
start of the session in question.

```mermaid
---
title: Delegated Application - RPC Request/Response & Claim/Proof
---
sequenceDiagram

    actor user as User
    actor app as Application (Delegated to Gateway)
    participant gw as Gateway
    participant pokt as Pokt Network
    participant sup as Supplier
    participant rpc as RPC Server


%% par Staking & Delegation
    gw-)pokt: Stake
    app-)pokt: Stake for service(s)
    app-)pokt: Delegate to gateway(s)
    sup-)pokt: Stake for service(s)
%% end

    loop Session N

        pokt--xpokt: Store current ring state for application

        gw-xpokt: Construct ring for signing (query application & delegates' public keys)
        sup-xpokt: Construct ring for verifying (query application & delegates' public keys)

        loop Every Request

            Note over user,rpc: ref: RPC Request/Response Relay (Delegated Application)

        end

    end

break Wait for session grace period to end
sup->>sup: 
    sup->>sup: Persist unclaimed & unproven session trees
end

loop Session+GracePeriod+1

loop Every application session from session N

Note over pokt,sup: ref: Supplier Claim/Proof

end

pokt--xpokt: Delete ring state for application

end

```
> **See**: [Legend > Sequence Diagram](#sequence-diagram)

## Legend

### Sequence Diagram
```mermaid
---
title: Sequence Diagram Legend
---
sequenceDiagram

    actor proto_part as Protocol Participant
    participant proto_actor as Protocol Actor

    proto_part->>+proto_actor: Protocol participant sends a synchronous <<message>> to protocol actor
    proto_actor--)-proto_part: Protocol actor returns a synchronous <<message>> to protocol participant

    loop Looped sequence

        proto_part-)+proto_actor: Protocol participant sends an asynchronous <<message>> to protocol actor
        proto_actor--)-proto_part: Protocol actor returns an asynchronous <<message>> to protocol participant

        Note over proto_part,proto_actor: ref: Interaction (other seq. diagram)

        proto_part--xproto_actor: An action of protocol participant updates on-chain state of protocol actor
        proto_part-xproto_actor: An action of protocol participant references on-chain state of protocol actor

        break Time gap
            proto_actor->>proto_actor: Protocol actor performs some independent action
        end

    end
```

## Reference Diagrams

### RPC Request/Response Relay (Sovereign Application)

### RPC Request/Response Relay (Delegated Application)

### 
