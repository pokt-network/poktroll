# Pocket 1.0 Utility Module Specification <!-- omit in toc -->

<p align="center">
    @andrewnguyen22 - Andrew Nguyen<br>
    @BenVanGithub - Shawn Regan<br>
    @olshansk - Daniel Olshansky<br>
    Version 1.0.1 - Jan 2023
</p>

- [1. Overview](#1-overview)
  - [1.1 The RPC Trilemma](#11-the-rpc-trilemma)
  - [1.2 Protocol Actors](#12-protocol-actors)
  - [1.3 Document Context](#13-document-context)
- [2. Requirements](#2-requirements)
- [3 Specification](#3-specification)
  - [3.1 Session Protocol](#31-session-protocol)
    - [3.1.1 Actor Selection (Dispatch)](#311-actor-selection-dispatch)
    - [3.1.2 RelayChain](#312-relaychain)
    - [3.1.3 GeoZone](#313-geozone)
    - [3.1.4 Actor Substitution](#314-actor-substitution)
    - [3.1.5 Rate Limiting](#315-rate-limiting)
    - [3.1.6 Interface](#316-interface)
  - [3.2 Servicer Protocol](#32-servicer-protocol)
    - [3.2.1 Staking](#321-staking)
    - [3.2.2 Network SLA (Service Level Agreement)](#322-network-sla-service-level-agreement)
    - [3.2.3 Report Cards \& Test Scores](#323-report-cards--test-scores)
    - [3.2.4 Volume Estimation](#324-volume-estimation)
    - [3.2.5 Salary Eligibility \& Distribution](#325-salary-eligibility--distribution)
    - [3.2.5 Pausing](#325-pausing)
    - [3.2.6 Parameter Updates](#326-parameter-updates)
    - [3.2.7 Stake Burning](#327-stake-burning)
    - [3.2.8 Unstaking](#328-unstaking)
  - [3.3 Fisherman Protocol](#33-fisherman-protocol)
    - [3.3.1 Election](#331-election)
    - [3.3.2 Staking](#332-staking)
    - [3.3.3 Sampling Protocol](#333-sampling-protocol)
    - [3.3.4 Incognito Sampling](#334-incognito-sampling)
    - [3.3.5 Salary Distribution](#335-salary-distribution)
    - [3.3.6 Test Score Submission](#336-test-score-submission)
    - [3.3.7 Parameter Updates](#337-parameter-updates)
    - [3.3.8 Pausing](#338-pausing)
    - [3.3.9 Unstaking](#339-unstaking)
    - [3.3.10 DAO Monitoring](#3310-dao-monitoring)
      - [3.3.10.1 Good Citizens Protocol](#33101-good-citizens-protocol)
  - [3.4 Application Protocol](#34-application-protocol)
    - [3.4.1 Staking](#341-staking)
    - [3.4.2 Parameter Updates](#342-parameter-updates)
    - [3.4.3 Unstaking](#343-unstaking)
    - [3.4.4 Stake Burning](#344-stake-burning)
  - [3.5 Gateway Protocol](#35-gateway-protocol)
    - [3.5.1 Gateway Responsibilities](#351-gateway-responsibilities)
    - [3.5.2 OAuth](#352-oauth)
    - [3.5.3 Application w/o Gateway](#353-application-wo-gateway)
    - [3.5.4 Application Delegation](#354-application-delegation)
    - [3.5.5 Application Servicing](#355-application-servicing)
    - [3.5.6 Gateway Registration](#356-gateway-registration)
    - [3.5.7 Gateway Unregistration](#357-gateway-unregistration)
    - [3.5.8 Application Undelegation](#358-application-undelegation)
  - [3.6 Validator Protocol](#36-validator-protocol)
    - [3.6.1 Staking](#361-staking)
    - [3.6.2 Block Rewards](#362-block-rewards)
    - [3.6.3 Pausing](#363-pausing)
    - [3.6.4 Stake Burning](#364-stake-burning)
    - [3.6.5 Parameter Updates](#365-parameter-updates)
    - [3.6.5 Unstaking](#365-unstaking)
  - [3.7 Account Protocol](#37-account-protocol)
    - [3.7.1 Account Structure](#371-account-structure)
    - [3.7.2 Send Transaction](#372-send-transaction)
    - [3.7.3 Pool](#373-pool)
  - [3.8 State Change Protocol](#38-state-change-protocol)
    - [3.8.1 Transaction](#381-transaction)
    - [3.8.2 Evidence](#382-evidence)
    - [3.8.3 Autonomous](#383-autonomous)
  - [3.9 Governance Protocol](#39-governance-protocol)
    - [3.9.1 Parameter Updates](#391-parameter-updates)
    - [3.9.2 DAO Treasury](#392-dao-treasury)
    - [3.9.3 Policing](#393-policing)
- [4. Launch Sequence](#4-launch-sequence)
  - [4.1 ProtoGateFish](#41-protogatefish)
  - [4.2 Castaway](#42-castaway)
  - [4.2 Fishermen](#42-fishermen)
  - [4.3 Feeder Fish](#43-feeder-fish)
  - [4.4 CastNet](#44-castnet)
- [5. Attack Vectors](#5-attack-vectors)
  - [5.1 Attack Category Types](#51-attack-category-types)
    - [5.1.1 Profit Seeking Passive](#511-profit-seeking-passive)
    - [5.1.2 Profit Seeking Active](#512-profit-seeking-active)
    - [5.1.3 Collusion Passive](#513-collusion-passive)
    - [5.1.4 Collusion Active](#514-collusion-active)
    - [5.1.5 Target Network Isolation (DDoS)](#515-target-network-isolation-ddos)
    - [5.1.6 Hacker](#516-hacker)
    - [5.1.7 Inflation, Deflation](#517-inflation-deflation)
    - [5.1.8 Mad Man Attacks](#518-mad-man-attacks)
      - [5.1.8.1 Mad Man Lazy](#5181-mad-man-lazy)
      - [5.1.8.2 Mad Man Hacker](#5182-mad-man-hacker)
      - [5.1.8.3 Masochistic Mad Man](#5183-masochistic-mad-man)
  - [5.2 Attack Examples](#52-attack-examples)
    - [5.2.1 Fisherman \<\> Servicer Collusion](#521-fisherman--servicer-collusion)
    - [5.2.2 Fisherman Assigning bad Servicer TestScores](#522-fisherman-assigning-bad-servicer-testscores)
    - [5.2.3 Fishermen falsifying Application Volume Metrics](#523-fishermen-falsifying-application-volume-metrics)
    - [5.2.4 Fisherman DDoS](#524-fisherman-ddos)
    - [5.2.5 Incognito Fisherman Identified](#525-incognito-fisherman-identified)
    - [5.2.6 Fisherman or Servicer register Applications for selfish self-dealing attacks](#526-fisherman-or-servicer-register-applications-for-selfish-self-dealing-attacks)
    - [5.2.7 Actors intentionally staking in “wrong” GeoZone](#527-actors-intentionally-staking-in-wrong-geozone)
- [6. Dissenting Opinions (FAQ)](#6-dissenting-opinions-faq)
- [References](#references)

## 1. Overview

This document describes Pocket Network’s Utility Module: an account based, state machine protocol that enables applications to permissionlessly access high quality decentralized Web3 services without the need of maintaining clients themselves.

### 1.1 The RPC Trilemma

Pocket Network aims to solve the RPC Trilemma:

1. **R**eliability: Uptime, geo-political redundancy, etc...
2. **P**erformance: QoS, latency, geographic decentralization, etc...
3. **C**ost: Web3 access, infrastructure maintenance, tokenomic incentivization & penalties etc...

![The RPC Trilemma](rpc_trilemma.png)

### 1.2 Protocol Actors

Pocket Network enables a Utilitarian economy that proportionally incentivizes or penalizes the corresponding infrastructure providers based on their quality of service. It is composed of the following actors:

- Staked **Applications** that purchase Web3 access over a function of volume and time
- Staked **Servicers** that earn rewards for providing Web3 access over a function of volume and quality
- Elected **Fishermen** who grade and enforce the quality of the Web3 access provided by **Servicers**
- Staked **Validators** responsible for maintaining safety & liveness of the replicated state machine
- Registered **Gateways** that can be optionally leveraged by **Applications** through delegated trust

```mermaid
flowchart TD
    subgraph Validators
        direction LR
        V1[Validator1]
        V[Validator ...]
        VN[Validator N]
    end
    V1 <--Propose/Vote--> V
    V1 <--Propose/Vote--> VN
    V <--Propose/Vote--> VN

    subgraph Blockchain
        direction LR
        B1[Block 1]
        B[Block ...]
        BN[Block N]
    end
    B1 --Transition--> B
    B --Transition--> BN

    subgraph Servicers
        direction TB
        S1[Servicer 1]
        SN[Servicer N]
    end

    subgraph Applications
        direction TB
        A1[Application 1]
        AN[Application N]
    end

    subgraph Gateways
        direction TB
        P1[Gateway 1]
        PN[Gateway N]
    end

    subgraph Fishermen
        direction TB
        F1[Fisherman 1]
        FN[Fisherman N]
    end

    Transactions --> Validators

    Validators --Validate State<br>Transition--> Blockchain

    Applications <--Delegated RPC--> Gateways
    Gateways <--App RPC--> Servicers
    Applications <--Trustless RPC--> Servicers

    Blockchain --Sync--> Gateways
    Blockchain --Sync--> Applications
    Blockchain --Sync--> Servicers
    Blockchain --Sync--> Fishermen

    Fishermen --Monitor--> Servicers

    classDef blue fill:#0000FF
    classDef brown fill:#A52A2A
    classDef red fill:#FF0000
    classDef yellow fill:#DC783D
    classDef acqua fill:#00A3A3
    classDef purple fill:#FF36FF

    class V1,V,VN blue
    class B1,B,BN brown
    class S1,SN yellow
    class P1,PN red
    class F1,FN acqua
    class A1,AN purple
```

### 1.3 Document Context

Readers of this document must keep in mind the following:

1. This living document is subject to change. Ongoing R&D will shape the specification until it is formalized and finished.
2. This document represents one stage of Pocket Network's evolution. Future iterations will aim to iterate on tokenomic incentives, permissionless Fisherman, Gateway incentives, etc.
3. This document should not be treated as a complete whitepaper. It is a specification of Utility Module components intended to drive the design and implementation of the technical specifications.
4. This document is not an academic paper. Formal proofs and verifications are absent and knowledge of background concepts is implicitly assumed.
5. This document does not outline implementation specific interfaces or details. Any interfaces presented are for illustrative purposes only.

## 2. Requirements

The specification must:

1. Enable Web3 access through a Utilitarian Economy leveraging native cryptocurrency capabilities
2. Account for both inflationary and deflationary economic scenarios
3. Incentivize actors within the protocol through competing offerings and economic penalties
4. Reward Servicers through a combination of quality of service, application demand volume, and services offered
5. Constrain service capacity through a combination of time, volume and cost
6. Account for gamification, collusion, and other attack vectors
7. Enable Applications to gain permissionless Web3 access without maintaining their own infrastructure
8. Enable Applications to optionally delegate trust to Gateways that ease the use of Web3 access

The current iteration of the specification must not necessarily:

1. Replace the safety guarantees of Applications maintaining their own infrastructure or using light clients
2. Enable permissionless registration of Fisherman actors on initial launch
3. Does not quantifiably model the underlying tokenomics of the system

## 3 Specification

The logical abstraction of the specification system is comprised of multiple sub-protocols listed below. Interfaces and diagrams presented are intended as aiding guidelines rather than definitive implementation details.

### 3.1 Session Protocol

A `Session` is a time-based mechanism used to regulate the various interactions (Web3 access, monitoring, etc) between protocol actors to enable the Utilitarian economy in a fair and secure manner. A single session may extend multiple Blocks as determined by `SessionBlockFrequency`.

#### 3.1.1 Actor Selection (Dispatch)

Under the Random Oracle Model, a Session can be seeded to deterministically select which group of actors will interact for some duration of time. This enables a random, deterministic and uniform distribution of Web3 access, provisioning and monitoring. It limits what work, and by whom, can be rewarded or penalized at the protocol layer.

The seed data for a session is an implementation detail that could be composed of multiple variables. It includes, but is not limited to, attributes such as `LatestBlockHash`, `Timestamp`, etc...

To start a new session, or retrieve the metadata for an existing / prior session, the Querier (e.g. Application, Client) can execute a request to any synched Full Node (protocol actor or not).

```mermaid
sequenceDiagram
    autonumber

    actor Q as Querier

    participant N as Full Node
    participant S as Session Interface

    Q->>N: Who are the Servicers &<br>Fisherman for this (new) Session?
    N->>S: seedData = (height, blockHash, geoZone, relayChain, app)
    S->>S: sessionKey = hash(transform(seedData))
    N->>S: servicerList = Ordered [publicKeys]
    S->>S: sessionServicers = pseudoRandomSelect(sessionKey, servicerList, maxSessionServicers)
    N->>S: fishermenList = Ordered [publicKeys]
    S->>S: sessionFishermen = pseudoRandomSelect(sessionKey, fishermenList, maxSessionFisherman)
    S->>Q: ([sessionServicers], [sessionFishermen])
```

For illustrative purposes, an example implementation of `NewSession` could be:

```go
func NewSession(sessionHeight, lastBlockHash, geoZone, relayChain, appPubKey) Session {
  key = hash(concat(sessionHeight, lastBlockHash, geoZone, relayChain, appPubKey))
  servicers = getClosestServicers(key, geoZone, numServicers)
  fishermen = getClosestFishermen(key, geoZone, numFishermen)
  return Session{sessionHeight, geoZone, relayChain, appPubKey, servicers, fishermen}
}
```

Note that a `timestamp` is explicitly not used to generate a new session because the process of generating a session (i.e. matching an application to servicers in some geozone at some height), must be a deterministic process executable by full nodes, light nodes and actors alike.

#### 3.1.2 RelayChain

A `RelayChain` is an identifier of the specified Web3 data source (i.e. a blockchain) being interacted with for that session.

For example, `0021` represents `Ethereum Mainnet` in Pocket Network V0.

#### 3.1.3 GeoZone

A `GeoZone` is a representation of a physical geo-location the actors advertise they are in.

For example, `GeoZone 0001` could represent `US East`, but alternative coordinate systems such as [Uber's H3](https://h3geo.org) or [PostGIS](https://postgis.net/) could be used as well.

There is no formal requirement or validation (e.g. IP verification) for an actor to be physically located in the GeoZone it registers in. However, crypto-economic incentives drive actors to be registered close to where they are physically located to receive and provide the best service possible.

The number of GeoZones an actor can stake for is limited to one to incentivize real geographic distribution. To quote @deblasis: "If an actor is everywhere, they're nowhere."

#### 3.1.4 Actor Substitution

Since a single Session extends multiple blocks, an actor could potentially send an on-chain transaction to exit (e.g. Unstake, Pause) prematurely. Any rewards for that Session for that actor are invalidated, and penalties may be applied. A replacement actor (e.g. a Servicer) will be found and dynamically added to the session in the closest following block.

#### 3.1.5 Rate Limiting

Rate limiting limits the amount of work (i.e. Web3 access) a Servicer can provide to an Application throughout the duration of a Session.

During each Session, the amount of POKT an Application has staked (see [Application Protocol](#34-application-protocol) for more details) is mapped to "Service Tokens" that represent the amount of work a Servicer can provide using the `SessionTokenBucketCoefficient` governance parameter. The [Token Bucket](https://en.wikipedia.org/wiki/Token_bucket) rate limiting algorithm is used to determine the maximum number of requests a Servicer can relay, and be rewarded for, thereby disincentivizing it to process relays for the Application once the cap is reached.

At the beginning of the session, each Servicer initializes: `AppSessionTokens = (AppStakeAmount * SessionTokenBucketCoefficient) / NumServicersPerSession`. When one of the Servicers in the session is out of session tokens, the Application can continue to use other Servicers until every they are all exhausted.

```mermaid
sequenceDiagram
  participant App as Application
  participant S1 as Servicer 1
  participant SN as Servicer N

  note over App, SN: Session Initialized

  loop
    App ->>+ S1: Relay
    alt AppSessionTokens > 0
      S1 ->> S1: AppSessionTokens -= 1
      S1 ->> App: Relay Response
    else
      S1 ->>- App: Error
    end
  end

  loop
    App ->>+ SN: Relay
    alt AppSessionTokens > 0
      SN ->> SN: AppSessionTokens -= 1
      SN ->> App: Relay Response
    else
      SN ->>- App: Error
    end
  end
```

The mechanism described above enables future iterations of the protocol where different types of request may vary the required number of `AppSessionTokens` per request.

#### 3.1.6 Interface

An illustrative example of the Session interface can be summarized as follows:

```go
type Session interface {
  NewSession(seeds ...interface{}) Session

  GetApplication() Application # The Application consuming Web3 access
  GetRelayChain() RelayChain   # The Web3 chain identifier being accessed this session
  GetGeoZone() GeoZone         # The physical geo-location where all the actors are registered
  GetSessionHeight() uint64    # The block height when the session started
  GetServicers() []Servicer    # The Servicers providing Web3 access
  GetFishermen() []Fisherman   # The Fisherman monitoring Web3 service
}
```

### 3.2 Servicer Protocol

A `Servicer` is a protocol actor that provisions Web3 access for Pocket Network Applications to consume. Servicers are the **supply** side of the Utilitarian Economy, who are compensated in the native cryptographic token, POKT, for their work: relaying RPC requests.

#### 3.2.1 Staking

In order to participate as a Servicer in Pocket Network, each actor is required to bond a certain amount of tokens in escrow while they are providing Web3 access. These tokens may be burnt or removed from the actor as a result of breaking the **Protocol’s Service Level Agreement**, a DAO defined set of requirements for the minimum quality of service.

Upon registration (i.e. staking), the Servicer is required to provide the network with sufficient information to be paired with Applications. This includes the `GeoZone` it is in, `RelayChain`(s) it supports, and `ServiceURL`, the Pocket API endpoint exposed where its services can be accessed. An optional `OperatorPublicKey` may be provided for non-custodial operation of the Servicer.

This registration message is formally known as the `StakeMsg`, and a Servicer can only start providing service once it is registered: the StakeMsg has been validated and included in the following block, thereby modifying the World State. An illustrative example can be summarized like so:

```go
type ServicerStakeMsg interface {
  GetPublicKey() PublicKey       # The public cryptographic id of the custodial account
  GetStakeAmount() BigInt        # The amount of uPOKT in escrow (i.e. a security deposit)
  GetServiceURL() ServiceURL     # The API endpoint where the Web3 service is provided
  GetRelayChains() []RelayChain  # The flavor(s) of Web3 hosted by this Servicer
  GetGeoZone() GeoZone           # The physical geo-location identifier this Servicer registered in
  GetOperatorPubKey() *PublicKey # OPTIONAL; The non-custodial pubKey operating this node
}
```

#### 3.2.2 Network SLA (Service Level Agreement)

Servicers are paid proportionally to how well their Relay responses meet the standards of the Network SLA. A `Relay` is an abstraction of a request/response cycle with additional security guarantees when interacting with Web3 resources. The quality of a service is measured by:

1. **Availability**: The Servicer's uptime
2. **Latency**: The Round-Trip-Time (RTT) of the the Servicer's response relative to when the request was sent
3. **Data Accuracy**: The integrity of the Servicer's response

Since the Fisherman may not necessarily be in the same GeoZone as the Application & Servicers, the Latency will onus will be on them to normalize the TestScores The Fisherman

#### 3.2.3 Report Cards & Test Scores

A `TestScore` is a collection of samples by a Fisherman of a Servicer, based on the SLA criteria outlined above, throughout the duration of a Session.

A `ReportCard` is the logical aggregation of multiple `TestScores` over an Actor's registration lifetime.

#### 3.2.4 Volume Estimation

The Application's Web3 usage volume is estimated through probabilistic hash collisions. This enables a concise proof of probabilistic volume, without requiring compute or memory intensive storage and aggregation. Similar to [Bitcoin's Difficulty](https://en.bitcoin.it/wiki/Difficulty), a `RelayVolumeDifficulty` governance parameter will be used to determine the "difficulty", and how relay counts must be estimated.

Each relay can be viewed as an independent Bernoulli Trial that is either a volume applicable relay or not. A geometric distribution can be built of the number of relays that need to be serviced until an applicable relay is made. For example, if a SHA256 hash algorithm is used and `RelayVolumeDifficulty` represents 3 leading zeroes, the hash of each `concat(SignedRelay, SignedRelayResponse)` above `0x000FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF` will not be applicable for relay rewards. However, any hash below the target will receive `RelayVolume` rewards proportional to the likelihood of the hash. For example, `0x000FFF...` would be estimate `RelayVolume` relays, `0x0000FF...` would e estimated to 16\*`RelayVolume`, etc...

A the end of each Session, the volume applicable relays are sent to the Fisherman for validation and salary distribution.

```mermaid
sequenceDiagram
    actor Client
    actor Servicer
    participant IS as Internal State
    participant ISS as Internal Storage
    participant Chain as Web3 Chain

    loop Repeats During Session
      Client ->> Client: Sign(Relay)
      Client ->> Servicer: Send(Relay)
      Servicer ->> +IS: Validate(Relay)
      IS ->> -Servicer: IsValid(Relay)
      Servicer ->> +Chain: Execute(Relay, RelayChainURL)
      Chain ->> -Servicer: RelayResponse = GetResponse(RelayChain)
      Servicer ->> Servicer: Sign(RelayResponse)
      Servicer ->> ISS: IfValid(Relay) -> Persist(Relay, RelayResponse)
      Servicer ->> Client: Send(RelayResponse)
    end

    loop Repeats Every Session End
      Servicer ->> +IS: GetSecretKey(sessionData)
      IS ->> -Servicer: HashCollision = SecretKey(govParams)
      Servicer ->> +ISS: RelaysThatEndWith(HashCollision)
      ISS ->> -Servicer: VolumeApplicableRelays
      Servicer ->> Fisherman: Send(VolumeApplicableRelays)
      Fisherman ->> Fisherman: Validate Relay & RelayResponse
    end
```

#### 3.2.5 Salary Eligibility & Distribution

A `ServicerSalary` is assigned to each individual Servicer based on their specific `ReportCard`, and is distributed every `SalaryBlockFrequency`. Salaries are distributed from the `TotalAvailableReward` pool, whose inflation is governed by Application volume of each `(RelayChain, GeoZone)` pair and scaled by the `UsageToRewardCoefficient` governance parameter.

A Servicer must accumulate `MinimumTestScoreThreshold` TestScores before it is eligible for salary distribution. A ReportCard can be viewed as a rolling average of the Servicer's performance, where TestScores are removed when either `TestScoreExpiration` is passed or the TestScore FIFO queue exceeds `MaxTestScores`.

Salary distribution is accomplished by aggregating the total volume estimated (see above) for a specific `(RelayChain, GeoZone)` pair (i.e. `TotalVolumeUsage`), multiplied by `UsageToRewardCoefficient`, and evenly divided into buckets per Servicer that exceed the minimum threshold (i.e. the `MinimumReportCardThreshold`). Each Servicer's reward is scaled proportionally to both their stake and their ReportCard. Tokens that are not allocated to a servicer are burnt.

For example, a 100% ReportCard results in zero burning of the `maxServicerReward`, while a 80% ReportCard results in 20% burning of the maxServicerReward. The rate of decrease continues linearly until the `MinimumReportCardThreshold` is reached. Below the MinimumReportCardThreshold no reward is given to prevent cheap Sybil attacks and freeloading nodes. Unstaking causes the Servicer's ReportCard to be cleared and start from scratch.

The following is pseudo-code to illustrate this business logic:

```go
// Called for each (relayChain, geoZone) pair every SessionBlockFrequency
func DistributeRewards(relayChain, geoZone, height):
  totalVolumeUsage = WorldState.RetrieveTotalVolumeEstimate(relayChain, geoZone, height)
  totalAvailableReward = totalVolumeUsage * GovParams.UsageToRewardCoefficient(height)

  allServicers = WorldState.RetrieveEligibleServicers(relayChain, geoZone, height)
  eligibleServicers = filterServicers(allServicers, GovParams.MinimumTestScoreThreshold(height))

  maxServicerReward = totalAvailableReward / len(eligibleServicers)

  for servicer := eligibleServicers {
      stake = WorldState.GetServicerStake(servicer, height)
      score = WorldState.GetReportCard(servicer, height)

      burnPercent = getBurnPercent(stake, score, height)
      burnAmount = burnPercent * maxServicerReward

      awardTokens(servicers, maxServicerReward - burnAmount)
      burnTokens(relayChain, geoZone, burnAmount)
  }
```

#### 3.2.5 Pausing

Servicers are able to gracefully pause their service (e.g. for maintenance reasons) without the need to unstake or face downtime penalization. In addition to an Operator initiated `PauseMsg`, Fishermen are also able to temporarily pause a Servicer if a faulty or malicious process is detected during sampling (see the [Fisherman Protocol](#33-fisherman-protocol) for more details). When a Servicer is paused, they are able resume service by submitting an `UnpauseMsg` after the `MinPauseTime` has elapsed. After a `UnpauseMsg` is validated and the World State is updated, the Servicer is eligible to continue providing Web3 service to Applications, and receive rewards.

#### 3.2.6 Parameter Updates

A Servicer can update any of the values in its on-chain attributes by submitting another `StakeMsg` while it is already staked. The only limitation is that it's `StakeAmount` must be equal to or greater than its currently staked value. In addition, the Servicer's historical QoS (TestScores, ReportCard, etc...) will be pruned from the state.

#### 3.2.7 Stake Burning

A Servicer's stake can be burnt in two situations:

1. A Servicer receives a `TestScore` below the `TestScoreBurnThreshold`
2. A Fisherman initiates a `PauseMsg` with the required evidence

#### 3.2.8 Unstaking

A Servicer is able to submit an `UnstakeMsg` to exit the network and remove itself from service. After a successful UnstakeMsg, the Servicer is no longer eligible to receive Web3 traffic from Applications or Fisherman. The original stake (i.e. deposit) is returned to the Servicer's custodial account after `ServicerUnbondingPeriod` has elapsed.

If a Servicer's stake ever falls below the `MinimumServicerStake` stake, the protocol automatically executes an UnstakeMsg on behalf of the custodial or operator accounts, subjecting the Servicer to the unstaking process described above.

### 3.3 Fisherman Protocol

A `Fisherman` is a protocol actor whose responsibility is to monitor and report the behavior and quality of Servicers' work. Their fundamental unit of work is to periodically sample the Servicers’s Web3 service, record the quality in a `TestScore`, and succinctly report the TestScore to the network in a `ReportCard`.

#### 3.3.1 Election

In the current version of the specification, Fishermen are not permissionless actors. The DAO must vote in each participant in order for the Fishermen to register themselves with the network on-chain. This requires Fisherman to advertise their identity and use their reputation as collateral against faulty and malicious behavior.

The Fishermen are bound by a DAO defined SLA to conduct incognito sampling services of the Servicers. Detailed requirements and conditions must be defined in the Fisherman SLA document to create an acceptable level of secrecy from the sampling server to ensure sample accuracy collection. It is important to note that the Fisherman `StakeMsg` message differs from other permissionless actors in Pocket Network because it is only valid if publicKey is permissioned through the DAO Access Control List (ACL) a prior.

```mermaid
stateDiagram-v2
    direction LR
    proposal: Propose Fisherman
    registration: Register Fisherman

    [*] --> proposal
    proposal --> proposal: Await DAO Acceptance
    proposal --> registration: Submit StakeTx
    registration --> registration: Await on-chain Validation
    registration --> [*]
```

#### 3.3.2 Staking

In addition to the Proof of Authority (PoA) process described above, Fisherman are also required to participate in a Proof of Stake (PoS) process by bonding (i.e. staking) a certain amount of POKT in escrow, on-chain, while they are providing the Fishermen services.

PoA and PoS are used to filter madmen adversaries who defy economic incentives in order to attack the network. Upon registration, a Fishermen must provide the necessary information to interact with the protocol, as illustrated in the interface below.

```go
type FishermanStakeMsg interface {
  GetPublicKey() PublicKey  # The public cryptographic id of the Fisherman custodial account
  GetStakeAmount() BigInt    # The amount of uPOKT in escrow (i.e. a security deposit)
  GetServiceURL() ServiceURL # The API endpoint where the Fishermen service is provided
  GetGeoZone() GeoZone     # The physical geo-location identifier this Fisherman is registered in
}
```

#### 3.3.3 Sampling Protocol

The Sampling Protocol is required to grade how each Servicer adheres to the network SLA criteria defined in the [Servicer Protocol](#32-servicer-protocol). Registered Fishermen are responsible for monitoring and periodically sampling Servicers during active Sessions.

`NumSamplesPerSession`, a governance parameter, defines how many samples a Fisherman is required to make to each Servicer during the duration of a sessions. To ensure a fair assessment across all Servicers in the session, the same request must be sent to all the Servicers at the same time during the time of evaluation. The frequency at which the samples are sent, as long as the `NumSamplesPerSession` quota is satisfied, should aim to be evenly distributed throughout the session. The performance of the Servicer will smooth out to the expected value over time through [The Law of Large Numbers](https://en.wikipedia.org/wiki/Law_of_large_numbers).

The data collected is:

1. **Availability**: Null sample if the Servicer is down or unresponsive
2. **Latency**: Round-Trip-Time (RTT) in milliseconds
3. **Data Consistency**: The accuracy of the data through Honest Majority consensus

```mermaid
sequenceDiagram
    autonumber

    actor App
    actor Fisherman
    actor Servicers

    loop Repeats During Session
        App ->> +Servicers: RPC Request
        Servicers ->> -App: RPC Response
        Fisherman ->> +Servicers: Incognito Sampling (RPC) Request
        Servicers ->> -Fisherman: RPC Response
    end

    Servicers ->> Fisherman: Reward Applicable Relay Proofs
    Fisherman ->> Fisherman: Sample Aggregation
    Fisherman ->> +World State: TestScore Txn
    World State ->> -World State: ReportCard Update

    Note over Fisherman, World State: Salary Distribution
    World State ->> Servicers: Reward For Service<br>(Based on ReportCard & Stake)
    World State ->> Fisherman: Reward For Sampling<br>(Based on Nullable Samples & Stake)
```

#### 3.3.4 Incognito Sampling

Servicers may bias to prioritize responding to requests from Fisherman in order to achieve a higher score. [Ring Signatures](https://en.wikipedia.org/wiki/Ring_signature) will be used in order to prevent identifying who signed the request: Fisherman, Application or Gateway.

```mermaid
flowchart
    subgraph Ring
        Application <--> F1["Fisherman 1"]
        F1 <--> F2["Fisherman N"]
        F2 <--> Gateway
        Gateway <--> Application
    end
    Ring --Signature--> Servicer
    Servicer--Validate Signature-->Servicer
```

Servicers may still be able to identify actors by aggregating and inspecting IP analytics, so other protocols (e.g. Tor, NYM, HOPR) or networking mechanism (IP rotations, VPNs, etc..) may be necessary to obfuscate the Fisherman as well.

#### 3.3.5 Salary Distribution

Fisherman salaries are dependant on the quantity and completeness of the `TestScoreMsg`s they send, but are agnostic to their contents with regard to the Servicers' Quality of Service.

**Report completeness** is based on the number of Nullable samples that have a value collected by the Fisherman. `NumSamplesPerSession` are expected to be performed by each Fisherman on each Servicer throughout a session.

Similar to the salary distribution algorithm described in the [Servicer Protocol](#32-servicer-protocol), Fisherman salaries are proportional to the estimated Application volume usage. However, Fishermen salary distribution is only based on the TestScoreMsg quantity and its consistence (i.e. # of non Null samples), as opposed to the actual test scores of the samples.

During each session, the Fisherman is expected to make `NumSamplesPerSession` of each active Servicer. The time at which they are made should be normally distributed throughout a session with some variance to avoid predictability. Servicers will timestamp and sign the requests to avoid forgery by the Fishermen.

If a Servicer does not respond, the sample is marked as a Null. Since Fisherman salaries are based off of Nullable samples with a value, they are incentivized to exhaust sampling attempts to retrieve the Servicer's score, up to a maximum of `NumSamplesPerSession`. If a Servicer exceeds `MaxNullableSamples` in a single Session, the Fishermen will opt to submit a PauseMsg of the unavailable Servicer (which partially burns the Servicer's stake), enabling the protocol to introduce a new Servicer into the Session with the following block. The PauseMsg action is limited up to `MaxFishermenPauses` per Session to prevent faulty Fishermen behavior and incentivizing Fisherman to keep the Session access as healthy as possible.

Individual Fishermen salaries are then calculated through a combination of Application volume usage in a certain `(RelayChain, GeoZone)` pair as well as the quantity of non-Null samples.

The following graph summarizes some of the high-level interactions described above.

```mermaid
graph LR
    F(Fisherman)
    B[Blockchain]
    RC((Report Card))
    S(Servicer)

    F <-."1. Sample<br>(collect TestScores)".-> S
    F -- "2. TestScore Txn<br>(from sampling)" --> B
    B -- "3. TestScore Aggregation<br>(nullable amples)" --> RC
    RC -- "4. Distribute Rewards<br>(based on work done)" --> S
    RC -- "5. Pay Salaries<br>(based on report completeness)" --> F
```

#### 3.3.6 Test Score Submission

On-chain test score submission follows an optimistic **Commit & Reveal** methodology to avoid excessive chain state bloat while maintaining sampling security. It is similar to the [Claim & Proof lifecycle](https://github.com/pokt-network/pocket-core/blob/staging/doc/specs/reward_protocol.md) designed in Pocket Network V0.

Fisherman `TestScoreMsg`s have a `TestScoreSubmissionProbability` requirement of being submitted, and `TestScoreProofProbability` requirement of needing to be proven. In practice, this means that while all Sessions are monitored, and all samples are made, only some are submitted on-chain, and a subset of those are formally proven. Similar to Session generation, this selection is done under the [Random Oracle](https://en.wikipedia.org/wiki/Random_oracle) model, so the Fishermen cannot predict which TestScores will be sampled, which need a commitment, and which need to be proven on-chain. Deterministic but non-predictable on-chain parameters (e.g. BlockHash at CurrentHeight+N) will be used to seed the selection.

```mermaid
pie title Test Scores Distribution
    "On-Chain Proven / Revealed" : 5
    "On-Chain Committed" : 20
    "Off-Chain" : 75
```

To reduce the amount of on-chain data, Fishermen are only required to prove a single Non-Null sample from their submitted `TestScoreCommitMsg` (i.e. claim) with a `TestScoreProofMsg` (i.e. reveal). This must be done before `MaxTestScoreProofDelay` blocks elapse.

Since the samples need to be normally distributed throughout the session, and the timestamp of each sample is signed by both the requester (i.e. Application/Gateway/Fisherman) and the responder (i.e. Servicer), the time of the sample selected using the seeded data is compared against the number of samples collected and the start time of the session sampling to verify its timestamp is within variance from the expected value.

```mermaid
---
title: Test Score Submission
---
classDiagram
    direction LR

    note for TestScoreCommitMsg "Used for rewarding & evaluating<br> Fisherman & Servicers"
    class TestScoreCommitMsg{
        Header: SessionData
        Height: 9001
        FirstSampleTime: 04:20 EST
        NumSamples: 5
        NumNullSamples: 2
        NulIndices: [2,3]
        Latencies: [p10, p25, p50, p75, p90]
        DataAccuracy: 69%
        EstimateRelaysServe: 1MM Relays
        Commit: SamplesRoot
    }

    note for TestScoreProofMsg "Used for optimistically<br>validating Fishermen work"
    class TestScoreProofMsg{
        Header: SessionData
        Sample: Sample2
        SignatureRequest: FishermanSig
        SignatureResponse: ServicerSig
        Proof: SampleProof
    }

    class Sample1{
        Time: 04:20 EST
        Result: [...]
    }
    class Sample2{
        Time: 04:21 EST
        Result: [...]
    }
    class Sample3{
        Time: 04:22 EST
        Results: NULL
    }
    class Sample4{
        Time: 04:23 EST
        Results: NULL
    }
    class Sample5{
        Time: 04:24 EST
        Results: [...]
    }

    TestScoreCommitMsg --> TestScoreProofMsg: N blocks later
    Sample2 --> TestScoreProofMsg: Index 1 (sample 2) selected

    Sample1 --> Sample2
    Sample2 --> Sample3
    Sample3 --> Sample4
    Sample4 --> Sample5
```

#### 3.3.7 Parameter Updates

A Fisherman can update any of the values in its on-chain attributes by submitting another `FishermenStakeMsg` while it is already staked. Parameter changes must be permissioned through the DAO ACL and the `StakeAmount` must be equal to or greater than its current value.

```go
type FishermenStakeMsg interface {
  GetPublicKey() PublicKey     # The public cryptographic ID of the Fisherman
  GetStakeAmount() BigInt       # May be modified to a value greater or equal to the current value
  GetServiceURL() ServiceURL    # May be modified
  GetGeoZone() GeoZone        # May be modified
}
```

#### 3.3.8 Pausing

A Fisherman can submit a `PauseMsg` to gracefully, and temporarily, remove themselves from service (e.g. for maintenance reasons).

Since Fishermen are DAO permissioned, a minimum of `MinActiveFisherman` must be registered and active at any point in time for a Fisherman to be able to pause or unstake. Downtime or inability to meet the DAO defined SLA may result in burning of the Fisherman's stake.

Similar to how Fishermen can send a `PauseMsg` on behalf of faulty Servicers, the DAO is able to pause faulty or malicious Fishermen as defined in [Pocket Network's Constitution](https://github.com/pokt-foundation/governance/blob/master/constitution/constitution.md).

A Fisherman can submit an `UnpauseMsg` after `MinPauseTime` has elapsed to resume monitoring services.

#### 3.3.9 Unstaking

Similar to unpausing, a Fisherman can permanently remove themselves from service when `MinActiveFisherman` are still active and registered. The stake will only be returned after `FishermenUnstakingTime` has elapsed and permissioned through the DAO to permanently end its services.

#### 3.3.10 DAO Monitoring

Enforcement of Fishermen behaviour and quality is an off-chain endeavor, undertaken by the DAO and crowd-sourced through the _Good Citizens Protocol_ of each protocol actor or community member. DAO monitoring consists of public Fishermen audits, statistical analysis, and incognito network actors that police interactions of Fishermen. The details, conditions, and limitations of DAO monitoring are defined further in the 1.0 Constitution.

In practice, the _Good Citizens Protocol_ acts as sanity checks for the network actors, developers, users and community to monitor the Fishermen. It is an opt-in, configurable module, that checks individual interactions against finalized on-chain data. Participants will be able to report suspicious, faulty, or malicious behavior of the Fishermen to off-chain data sites which are analyzed and filtered up to the DAO and public. Individual Fishermen burns and Good Citizen bounties are determined by the DAO and defined in the 1.0 Constitution.

##### 3.3.10.1 Good Citizens Protocol

The _Good Citizens Protocol_ is not a "real" protocol. It is a social committment that is often the result of tools being built and focus being shifted to areas where work is being done and money is being made. Since a permissionless network will be earning rewards based on the, initially, permissioned set of Fisherman, low rewards values are likely be noticed and floated to the surface leading to the uncovering of misbehaviour, sooner or later.

### 3.4 Application Protocol

An `Application` is a protocol actor that consumes Web3 access from Pocket Network Servicers. Applications are the **demand** side of the Utilitarian Economy, who are paying in the native cryptographic token, POKT, for the utility provided by the decentralized and permissionless network.

#### 3.4.1 Staking

In order to participate as a Application in Pocket Network, each actor is required to bond a certain amount of tokens in escrow while they are consuming the Web3 access. Upon registration, the Application is required to provide information necessary to create applicable Sessions (GeoZone(s), RelayChain(s), etc...). The Application will also be able to specify `NumServicers` (bounded by `MinServicersPerSession` and `MaxServicersPerSession`) to specify the number of servicers it would prefer, if available, for that session.

It is important to note that the bond amount is directly proportional to `MaxRelaysPerSession`, which limits the number of requests each App can make per Servicer per Session as explained in the [Session Protocol](#31-session-protocol).

```go
type ApplicationStakeMsg interface {
  GetPublicKey() PublicKey      # The public cryptographic id of the Application
  GetStakeAmount() BigInt       # The amount of uPOKT in escrow (i.e. a security deposit)
  GetRelayChains() []RelayChain # The flavor(s) of Web3 hosted by this Application
  GetGeoZone() GeoZone          # The physical geo-location identifier this Application registered in
  GetNumServicers() uint8       # The number of Servicers requested per session
}
```

#### 3.4.2 Parameter Updates

An Application can update any of the values in its on-chain attributes by submitting another `ApplicationStakeMsg` while it is already staked. Any parameter changes are permissible as long as the `StakeAmount` is equal to or greater than its current value.

```go
type ApplicationStakeMsg interface {
  GetPublicKey() PublicKey     # The public cryptographic ID of the Fisherman
  GetStakeAmount() BigInt       # May be modified to a value greater or equal to the current value
  GetRelayChains() []RelayChain # May be modified
  GetGeoZone() GeoZone          # May be modified
  GetNumServicers() uint8       # May be modified
}
```

#### 3.4.3 Unstaking

An Application is able to submit an `UnstakeMsg` to exit and remove itself from the network. After a successful UnstakeMsg, the Application is no longer eligible to consume Web3 traffic from Servicers. After the `ApplicationUnstakingTime` unbonding time elapses, the remaining stake is returned to the custodial account.

#### 3.4.4 Stake Burning

Application stake burn is a necessary mechanism to ensure economic equilibrium at network maturity by balancing POKT inflation and deflation.

`AppBurnPerSession` is a governance parameter that will dictate the amount of POKT burnt for every session an Application initiates, and `AppBurnPerRelay` will govern the amount of POKT burnt based on the amount of work (i.e. number of relays serviced) Servicers provided to the Application throughout the session. Given that the Application's stake will decrease with every session, new rate limiting parameters, dictated via `MaxRelaysPerSession`, will be set at the start of each new session.

As of updating this document, these governance parameters are expected to be 0 at the time of launching the next version of the network. More detailed tokenomic models will follow in future iterations of the specifications.

### 3.5 Gateway Protocol

A `Gateway` is a permissionless protocol actor to whom the Application can **optionally** delegate on-chain trust in order to perform off-chain operations.

#### 3.5.1 Gateway Responsibilities

Pocket Network's Utilitarian Economy incentivizes data redundancy in a multi-chain ecosystem, with cheap, accessible and highly available multi-chain access. Depending on the level of trust, or lack thereof, an Application can optionally use a Gateway for various Pocket-specific operations such as, but not limited to, session dispatching or request signing.

Delegation also enables free market off-chain economics where additional features, guarantees or payments can be made. This could, for example, include a contractual agreement between Applications and Gateways to execute [Client Side Validation](https://forum.pokt.network/t/client-side-validation/148) with every Nth request. It could also enable L2 services, such as data indexing, that are outside the scope of the Pocket ecosystem, but are closely related to the utility it provides.

Applications that requires just-in-time full data integrity guarantees may opt out of delegating to Gateways and operate in a completely permissionless manner. This may require them to maintain their own infrastructure (i.e. synching a full/light Pocket Node). Even with delegation, an Application would be able to continue operating permissionlessly (via a full or light node) as the two are not mutually exclusive.

#### 3.5.2 OAuth

[OAuth](https://oauth.net) is an open (Web2) protocol that authorizes clients or 3rd parties to gain access to restricted resources. It can be summarized via following flow:

```mermaid
sequenceDiagram
    actor U as User
    participant C as Client<br>(e.g. Smartphone App)
    participant S as Authorization Server<br>(e.g. Google)
    participant R as Resource Server<br>(e.g. Email)

    U ->> C: Request access<br>to protected resource
    C ->> +S: Request authorization
    S -->> U: Prompt for authorization
    U -->> S: Grant authorization<br>(Username & Password)
    S ->> C: Return authorization_code
    C ->> S: Exchange authorization_code for access_token
    S ->> -C: Return access_token
    C ->> +R: Request protected resource
    R -->> -C: Return protected resource
```

For the sake of simplicity, we are omitting `refresh_token` related considerations.

Some parallels can be drawn between existing centralized, trusted and permissioned systems relative to Pocket's Utilitarian Economy:

- The `Client` remains as the `Client`
- The `Application` is the `User`
- The `Application` is a one-time `Authorization Server`
- The `Gateways` is an ongoing `Authorization Server`
- The `Servicer` is the `Resource Servicer`
- The `Fisherman` is a separate monitoring party overlooking the `Resource Servicer` most often owned by the `Authorization Server`

#### 3.5.3 Application w/o Gateway

An Application that chooses to operate without a Gateway is responsible for dispatching sessions and signing RPC requests on its own. To do so, it will need to maintain a Pocket Full Node or a Pocket Light Client.

```mermaid
sequenceDiagram
    actor AC as Application / Client
    actor LN as Local Pocket Node / <br>Local Light Client
    actor SN as Servicer

    AC ->> +LN: StartSession()
    LN ->> -AC: SessionData([ServicerIDs], ...)

    loop Session Duration
        AC ->> AC: Sign Request

        AC ->>+ SN: SignedRequest
        SN ->> SN: Validate Signature<br>& Session Limits
        SN ->> SN: Handle Request<br>& Sign Response
        SN ->>- AC: SignedResponse

        AC ->> AC: Process response
    end
```

#### 3.5.4 Application Delegation

An Application that chooses to delegate trust to a Gateway will need to submit a one-time `DelegateMsg` transaction to delegate trust from the Application to the Gateway. It must include the PublicKey of the Gateway and be signed by the Application.

```mermaid
sequenceDiagram
    actor A as Application
    participant WS as World State
    actor G as Gateway

    A ->> A: Prepare Delegate Message
    A ->> A: Sign Request

    A ->>+ WS: Delegate(GatewayPubKey)
    WS ->>- A: ok
```

The following message will need to be signed by the Application's PrivateKey in order for it to be valid and committed to the world state.

```go
type DelegateMsg interface {
  GetApplicationPublicKey() # The cryptographic ID of the Application
  GetGatewayPublicKey()     # The cryptographic ID of the Gateway
}
```

Once committed, the Application can be serviced on behalf of the Gateway. Though an Application can delegate to multiple Gateways simultaneously, the rate limiting for each session still remains at the Servicer level.

#### 3.5.5 Application Servicing

When an Application chooses to start a new session, the Gateway is responsible for dispatching the `StartSession` request using on-chain and use an off-chain mechanism (e.g. AccessTokens) to service the Application. Throughout the duration of the session, validation and communication between the Application and Gateway are done using off-chain mechanisms, which are outside the scope of this document.

[Ring Signatures](https://en.wikipedia.org/wiki/Ring_signature) will be used in order to allow both the Application and the Gateway to sign the Relay.

```mermaid
flowchart
    subgraph Ring
        Application <--> P1["Gateway 1"]
        P1 <--> P2["Gateway 2"]
        P2 <--> Application
    end
    Ring --Signature--> Servicer
    Servicer--Validate Signature-->Servicer
```

Similar to to the [incognito sampling section of the Fisherman Protocol](#334-incognito-sampling) section, Ring Signatures enable the Servicer to validate the signed request. This enables permissioned (w/ a Gateway) and permissionless (w/o a Gateway) operations to co-exist, without being mutually exclusive, and without the Servicer needing knowledge of the Application's current mode of operation.

```mermaid
---
title: Signature Validation By Servicer
---
stateDiagram-v2
    state "Get gateways the App<br>delegated to: [P1, P2]" as getGateways
    state "Is relay signed by one of:<br>Application, Gateway1, Gateway2?" as sigCheck

    state "Valid (should service relay)" as Valid
    state "Invalid (do not service relay)" as Invalid

    [*] --> getGateways
    getGateways --> sigCheck

    sigCheck --> Valid: Yes
    sigCheck --> Invalid: No
```

Servicer's are incentivized to respond to any valid relay since it is applicable for reward distribution. The session tokens used for rate limiting by the Servicer will come out of the same bucket as described in the [rate limiting algorithm](#315-rate-limiting) regardless of who in the ring signed the request.

```mermaid
sequenceDiagram
    actor A as Application
    participant WS as World State
    actor G as Gateway
    actor S as Servicer

    A ->>+ G: StartSession
    G ->> G: StartSession
    G ->>- A: AccessToken

    loop Session Duration
        A ->> +G: Request(AccessToken)
        G ->> G: Sign Request
        G ->> +S: SignedRequest
        S ->> S: Validate Signature<br>& Session Limits
        S ->> S: Handle Request<br>& Sign Response
        S ->> -G: SignedResponse
        G ->> G: ** Gateway specific features ** <br>(altruist, check, challenge, proof, etc...)
        G ->> -A: Response
    end
```

#### 3.5.6 Gateway Registration

Registration differs from staking in the sense that the pubKey is known but there are no economic benefits/penalties in this stage of the protocol's progression.

The Gateway must register on-chain in order for the Servicer to accept its signature as part of the ring. Future versions of the protocol may include on-chain rewards or penalties for the Gateway, but the current iteration will incentivize Gateways to provide a high quality, highly trusted service through free market economics.

When staking, the Gateway must bond a certain amount of POKT to be able to participate in the network. The governance parameter, `StakePerAppDelegation` limits the number of Applications that can delegate to it, and it is the Gateway's responsibility to increase its stake as the number of Applications that trust it grow.

For example, if `StakePerAppDelegation` is 100 POKT and the Gateway has staked 1000 POKT, a transaction by the 11th Application to delegate to it will be rejected until the stake is increased appropriately. However, if `StakePerAppDelegation` is 0 POKT, all Gateways, which are permissionless actors can have an unbounded number of Applications delegate to them.

If `StakePerAppDelegation` changes such that a Gateway cannot support the existing numbers of delegating apps, they are all legacied in to continue operating as normal. However, new applications cannot delegate to the Gateway until the stake is sufficiently increased.

```go
type GatewayStakeMsg interface {
  GetPublicKey() PublicKey   # The public cryptographic id of the Gateway account
  GetStakeAmount() BigInt     # The amount of uPOKT in escrow (i.e. a security deposit)
  GetServiceURL() ServiceURL  # The API endpoint where the Gateway service is provided
}
```

#### 3.5.7 Gateway Unregistration

A Gateway is able to submit an `UnstakeMsg` to exit and remove itself from the network. After a successful UnstakeMsg, the Gateway is no eligible sign relays on behalf of an Application. On-chain delegation from existing Applications will be removed from the world state. After the `GatewayUnstakingTime` unbonding time elapses, the remaining stake is returned to the Gateway's address.

#### 3.5.8 Application Undelegation

If a staked Application wants to stop using a Gateway, and prevent the Gateway from further signing relays on its behalf, it would simply submit an on-chain `UndelegateMsg`. Further relays signed by the Gateway on behalf of the Application would be rejected by the Servicers.

```go
type UndelegateMsg interface {
  GetApplicationPublicKey() # The cryptographic ID of the Application
  GetGatewayPublicKey()     # The cryptographic ID of the Gateway
}
```

### 3.6 Validator Protocol

A `Validator` is a protocol actor whose responsibility is to securely validate and process state transitions through transactions. It does so through Byzantine Fault Tolerant consensus. See the accompanying [Consensus Specification](../consensus/README.md) for full details on its internals.

Though most of their functional behavior is described in the external [Consensus Specification](../consensus/README.md), a state specific Validator Protocol is needed to allow a dynamic Validator set and provide the necessary incentive layer for their operations.

#### 3.6.1 Staking

Validators registration is permissionless. In order to participate in the network as a Validator, each actor is required to bond a certain amount of tokens in escrow while they are validating state transitions. These tokens are used as collateral to disincentive byzantine behavior.

Upon registration, a Validator must inform the network of how many tokens they are staking and the public endpoint where the Validator API is securely exposed. In addition to the required information, an optional OperatorPublicKey may be provided for non-custodial operation of the Validator. This registration message is formally known as the `ValidatorStakeMsg`:

```go
type ValidatorStakeMsg interface {
  GetPublicKey() PublicKey        # The public cryptographic id of the custodial account
  GetStakeAmount() BigInt         # The amount of uPOKT in escrow (i.e. a security deposit)
  GetServiceURL() ServiceURL      # The API endpoint where the validator service is provided
  GetOperatorPubKey() *PublicKey  # OPTIONAL; The non-custodial pubKey operating this node
}
```

#### 3.6.2 Block Rewards

The Block Producer for each height (i.e. a single Validator) will be rewarded `BlockProposerAllocation` % of the total `TransactionFees` and `RelayRewards` held in the new block.

A single Validator's stake relative to the cumulative total of all Validators' stake is directly proportional to the likelihood of them being selected as a block producer. The external Consensus Specification outlines how block producers are selected for each height in greater detail.

The net `BlockProducerReward` is derived from the `TotalBlockReward` by subtracting the `DAOBlockReward` amount. Once a proposal block is finalized into the blockchain, the `BlockProducerReward` is sent to the custodial account.

#### 3.6.3 Pausing

Validators are able to gracefully pause their service (e.g. for maintenance reasons) without the need to unstake or face downtime penalization.

#### 3.6.4 Stake Burning

Validators are paused from service by the network if byzantine behaviors are detected. Some examples include:

- Not producing blocks when expected to
- Not signing blocks
- Signing against the majority
- Double signing blocks

Anytime an automatic removal of a Validator occurs, a burn proportional to the violation is applied against the Validator stake. The conditions, limitations, and severity of the Byzantine Validator burns are proposed and voted on by the DAO.

If a Validator is `paused`, they are able to reverse the paused state by submitting an `UnpauseMsg` after the `MinValidatorPauseTime` has elapsed. After a successful UnpauseMsg, the Validator is once again eligible to execute Validator operations.

#### 3.6.5 Parameter Updates

A Validator can update any of the values in its on-chain attributes by submitting another `StakeMsg` while it is already staked. The only limitation is that it's `StakeAmount` must be equal to or greater than its currently staked value.

#### 3.6.5 Unstaking

A Validator is able to submit an `UnstakeMsg` to exit the network and remove itself from Validator Operations. After a successful UnstakeMsg, the Validator is no longer eligible to participate in the Consensus protocol.

After `ValidatorUnstakingTime` elapses, any stake amount left is returned to the custodial account. If a Validator stake amount ever falls below the `MinimumValidatorStake`, the protocol automatically executes an UnstakeMsg on behalf of the node, subjecting the Validator to the unstaking process.

### 3.7 Account Protocol

An `Account` is a structure that maintains the ownership of POKT via a mapping from an `Address` to a `Balance`.
The summation of all balances of all accounts in the network equals the `TotalSupply`.

#### 3.7.1 Account Structure

```go
type Account interface {
  GetAddress() Address # The cryptographic ID of the account
  GetBalance() Big.Int # The amount of uPOKT
}
```

#### 3.7.2 Send Transaction

The only state modification an Account may execute is a `SendMsg` of POKT from one account to another.

```go
type SendMsg interface {
  GetFromAddress() Address # Address of sender Account; the signer
  GetToAddress() Address # Address of receiver Account
  GetAmount() Big.Int # The number of uPOKT transferred
}
```

#### 3.7.3 Pool

A Pool is a special type of Account that is completely autonomous and owned by the network. POKT can be burnt, minted and transmitted to a Pool at the protocol layer at the time of state transition (after a block is validated) without explicit transactions. Pools can be used for tasks such as fee aggregation, reward distribution and other operations based on state machine changes.

```go
type Pool interface {
  GetAccount() Account # The cryptographic ID of the account
  GetName() string     # The name of the pool
}
```

The list of pools includes, but is not limited to, the `DAO`, `FeeCollector`, `AppStakePool`, `ValidatorStakePool`, `ServicerStakePool`.

### 3.8 State Change Protocol

There are two categories of state changes in Pocket Network that may be included in a block:

1. Transactions - state changes initiated and signed by any account with or without a balance
2. Evidence - state changes and/or signals initiated and signed by registered/staked protocol actors
3. Autonomous - operations completed based on the results of the lifecycle of validated and finalized blocks

#### 3.8.1 Transaction

Transactions are discrete state change operations executed by actors or accounts. The structure of a Transaction includes, at a minimum:

- **Payload**: A command structured in the form of a Message (e.g. StakeMsg, PauseMsg, SendMsg, etc..)
- **Dynamic fee**: A fee to incentivize the block producer to include the Transaction in a block and deter sybil attacks
- **Authenticator**: A digital signature
- **Entropy**: A Nonce (i.e. a random number) to prevent replay protection

```go
type Transaction interface {
  GetPublicKey() PublicKey  # Cryptographic identifier of sender account; must be the signer
  GetSignature() Signature  # Digital signature of the transaction
  GetMsg() Message          # The payload of the transaction; Unstake, TestScore, etc.
  GetFee() Big.Int          # The number of tokens (uPOKT) used to incentivize and pay for the execution
  GetNonce() Big.Int        # Entropy used to prevent replay attacks; upper bounded
}
```

#### 3.8.2 Evidence

Evidence is similar to Transactions in creation, structure and handling, but its production and affects are limited based on the actor's role.

For example, only `Validators` are eligible to submit and are affected by `DoubleSign` evidence. Only `Servicers` are effected by the results of a `ClientSideChallenge` evidence. Only `Fishermen` are effected by the results of evidence that may challenge by the results of its TestScores (e.g. `RegradeServicer`)`.

Evidence is a protection mechanism against faulty or malicious consensus participants, but may extend to other protocol actors as well. The full list of types of Evidence will be defined over time, but will often result in the burning of the actor's stake if proven true.

```go
type Evidence interface {
   GetMsg() Message        # Payload of the evidence; DoubleSign, TimeoutCert, ClientSideChallenge, etc.
   GetHeight() Big.Int     # Height at which the evidence was collected and submitted
   GetAuth() Authenticator # Authentication of evidence, can be a digital signature or certificate
}
```

#### 3.8.3 Autonomous

Autonomous state change operations are performed by the protocol according to specific lifecycle triggers linked to consensus. These operations include rewarding block producers, distributing relay rewards, etc.

### 3.9 Governance Protocol

Though most off-chain governance specification is included in [Pocket Network's Constitution](https://github.com/pokt-foundation/governance/blob/master/constitution/constitution.md), the Utility Module’s Governance Protocol defines how the DAO is able to interact with the on-chain protocol on a technical level.

#### 3.9.1 Parameter Updates

The Access Control List (ACL) is the primary mechanism that enables live parameter configuration. Specifically, the ACL maintains the "feature switches" that allow active modification of the behavior of the protocol without forks. The ACL also maintains the account(s) permissioned to modify said parameters. A value can be modified by a `ParamUpdateMsg` from the permissioned owner of that parameter.

```go
type ParamChangeMsg interface {
  GetAddress() Address  # Address of sender & signer; must be permissioned through ACL
  GetParamName() String # The name of the parameter being updated
  GetValue() any        # The new value of the parameter being modified
  GetMemo() *String     # Explanation (or URL) of why the change was made
}
```

#### 3.9.2 DAO Treasury

In addition to parameter changes, the DAO is able to burn or transfer from a specified DAO Module Account. In practice, this Module Account is used as the on-chain Treasury of the DAO, the limitations and specifics of which are maintained in the DAO 1.0 Constitution document.

```go
type DAOTreasuryMsg interface {
  GetSrcAddress() Address  # The sender of the funds; must be permissioned through ACL
  GetDstAddress() *Address # OPTIONAL; the receiver of the funds if applicable
  GetOperation() DAOOp     # The identifier of the operation; burn or send
  GetAmount() Big.Int      # The operation is executed on this amount of tokens
  GetMemo() *String        # Explanation (or URL) of why the reasoning for the fund transfer
}
```

#### 3.9.3 Policing

As the only permissioned actors in the network, Fishermen are subject to individual burns, pauses, or removals initiated by the DAO. Usages of this message type are a result of the off-chain monitoring mechanisms described in the Fisherman Protocol section of the document. The specifics and limitations of usage of this message type is detailed in the DAO 1.0 Constitution.

```go
type PolicingMsg interface {
  GetAddress() Address # Address of sender & signer; must be permissioned on ACL
  GetPoliced() Address # Address of the policed actor
  Operation() DAOOp    # Identifier of the operation; burn, pause, remove, etc
  GetAmount() Big.Int  # Amount of tokens (if applicable)
  GetMemo() *String    # Explanation (or URL) of the issue
}
```

## 4. Launch Sequence

In order to fully understand Pocket Network 1.0 and its place in the project's maturity, a rollout context is a prerequisite.

### 4.1 ProtoGateFish

During the development & development of a v1 TestNet:

- The Fisherman & Gateway models will be will be prototyped and live-tested
- Major Servicer node runners of Pocket Network V0 may have the option to participate as Fisherman and/or Gateway actors

### 4.2 Castaway

Upon the initial launch of Pocket Network v1 MainNet:

- The DAO will need to approve at least one PNI owned Fisherman

  - The publicKey of the Fisherman will be included in the re-genesis file

- PNI will need to to register at least one Gateway

  - The publicKey of the Gateway will be included in the re-genesis file
  - The community and/or DAO will choose the initial value for `StakePerAppDelegation`
  - Other actors can register their own Gateways shortly after launch

- Pocket Network scalability issues will be resolved
- Permissionless Applications will be enabled
- The economy will enable Network participants to operate their own Gateways
- `AppBurnPerSession` and `AppBurnPerRelay` will be set to 0

### 4.2 Fishermen

- Fisherman governance parameters will be tuned
- Additional Fisherman will be approved by the DAO

### 4.3 Feeder Fish

- Application stake parameters (`AppBurnPerSession` and `AppBurnPerRelay`) will be tuned

### 4.4 CastNet

- A specification for Fisherman Gateway scores will be designed and developed

## 5. Attack Vectors

_NOTE: This section does not exhaust all attack vectors considered, rather details the ones that were deemed important._

### 5.1 Attack Category Types

#### 5.1.1 Profit Seeking Passive

**Primary goal**: Long term. Increase the total value of the attacker's assets.

**Secondary goals**: Avoiding detection and damage to personal or network reputation.

**AKA**: Don’t kill the goose that lays the golden eggs.

#### 5.1.2 Profit Seeking Active

Short term. Increase the total value of the attacker's assets and sell it before the network/DAO can notice and react.

**AKA**: Take the money and run.

#### 5.1.3 Collusion Passive

A single active attacker relies on tacit cooperation from a second party that benefits from not reporting the attack.

**AKA**: Don’t ask. Don’t tell.

#### 5.1.4 Collusion Active

Two active players in two different parts of the economic system seek to circumvent the system by actively submitting fraudulent information and/or blocking attempts to punish, penalize or burn the malfeasant actor(s).

**AKA**: The master plan.

#### 5.1.5 Target Network Isolation (DDoS)

Profit seeking attacks seek to reduce the effectiveness and/or number of competitors, thereby increasing the share of rewards received.

**AKA**: Kill the competition.

#### 5.1.6 Hacker

An attack which requires creation, modification and upkeep of the core software in a private repository w/ deep protocol expertise.

**AKA**: Mr. Robot

#### 5.1.7 Inflation, Deflation

An attack whose net effect is an increase or decrease in the rate of production of new tokens.

In most cases, these are likely to be **Mad Man Attacks** because any benefit/deficit created will be distributed throughout the entire network.

**AKA**: Make It Rain.

#### 5.1.8 Mad Man Attacks

Non-profit seeking seeking attacks/behaviors which cost more to perform than can reasonably be expected to benefit the attacker. Usually results in damage to the network's reputation.

**AKA**: Kill 'em all.

##### 5.1.8.1 Mad Man Lazy

Attack does not require significant time, energy or resources.

**AKA**: What happens when I push this button?

##### 5.1.8.2 Mad Man Hacker

Non-profit seeking attack which requires creation, modification and upkeep of the core software in a private repository.

##### 5.1.8.3 Masochistic Mad Man

Any attack which, although possible, makes no sense because you could accomplish the same thing without doing all that work.

**AKA**: Rube Goldberg Attack.

### 5.2 Attack Examples

<!-- TODO(olshansky): Review & Update -->

#### 5.2.1 Fisherman <> Servicer Collusion

**Attack Vectors**: Profit Seeking Passive; Collusion Active; Hacker

A Fisherman colluding with one or more Servicers and is in possession of their keys is able to falsify all aspects of that Servicer's report card. Therefore, all of his colluding node partners get A+ report cards and resultantly larger paychecks.

This is the “big one”. It is the primary reason that Fisherman (at this stage) require DAO Approval. It is why off chain data logs, good-citizen reporting and DAO oversight exist. It is also the reason that Fishermen require large stake/deposit, as well as increased destaking period and delayed payments (See attached spreadsheet of projected collusion ROI based on such factors as Node Percentage, Risk Rate, etc.)

The attack is easy to describe, but not easy to perform because a rather large body of work has gone into making sure that it is extremely difficult to perform, a very small payoff, and extremely costly to get caught.

#### 5.2.2 Fisherman Assigning bad Servicer TestScores

**Attack Vectors**: Collusion passive; Hacker; Mad Man Lazy; Inflation/Deflation

If a Fisherman falsifies certain aspects of a node’s report card, he can lower that node’s proper payment share (direct attack) and increase the relative profit of all non-attacked nodes (indirect benefit if Fisherman is also a Servicer owner)

This attack is a lot of work with very little reward. The “excess” POKT does not get distributed to the non-attacked nodes. It gets burned. Therefore the benefit to non-attacked nodes is only a relative gain in terms of the overall network inflation rate. This is a highly detectable attack and the victims are highly motivated to report it. Bottom line here is: We’re talking about a highly motivated hacker, who is also Mad Man Lazy.

#### 5.2.3 Fishermen falsifying Application Volume Metrics

**Attack Vectors**: Mad Man Lazy

There is no financial incentive for a Fisherman to report Application volume higher or lower than actual. Servicers are incentivized to check for under reporting. There is no specific disincentive for over reporting. Other than (of course) losing all of your stake and reputation.

#### 5.2.4 Fisherman DDoS

**Attack Vectors**: Mad Man

No one benefits by DDOSing a Fisherman. The relay, dispatch, service and blockchain processes are not dependent on Fishermen. The attack does not change node report cards. It only reduces overall network inflation. The Fisherman loses money, but no one gets the excess.

#### 5.2.5 Incognito Fisherman Identified

**Attack Vectors**: Lazy Mad Man, Hacker

Fishermen act in “incognito” fashion purely as a deterrent to a particular theoretical publicity seeking attack called Mad Man Blogger. Neither honest, nor dishonest nodes gain any advantage by identifying a particular relay request as belonging to a Fisherman. You can only provide your best service, it’s not possible to provide “better” service to a Fisherman than you already provide. However, the Mad Man Blogger could (if he chose to and if he successfully identified all Fishermen) provide service to only Fishermen and not to applications. We consider this an edge case attack in which the attacker has made a significant investment in POKT and in backend node infrastructure solely for the purpose of “proving” that the system can be gamed. Therefore, the contract with the DAO which Fishermen agree to, requires periodic changing of IP address and reasonable efforts to keep it unknown by other actors in the ecosystem.

#### 5.2.6 Fisherman or Servicer register Applications for selfish self-dealing attacks

**Attack Vectors**: Inflation Attack, Mad Man Lazy

**Re Fishermen spamming via owned app:**

There is no economic benefit to Fishermen from this activity as their payment is independent of application usage. One could argue that the Fisherman will benefit from the increase in supply caused by spamming the network this way. However, any inflation attack which benefits the majority of the network to a greater degree than the person who pays for and performs the attack is non-profit seeking.

**Re Servicer spamming via owned app:**

This activity is an inflation attack which is not profitable to the attacker.

#### 5.2.7 Actors intentionally staking in “wrong” GeoZone

It is economically advantageous for actors to stake within their true GeoZone as Session actor pairings are generated specifically for the GeoZone registered. Specifically, Applications will receive worse QOS and worse fidelity of Fishermen QOS monitoring by proxy as the Servicers are farther away but still within the same GeoZone as the Fishermen. Servicers are the same way, as once a GeoZone is mature the Servicers in other GeoZones are no longer competitive.

## 6. Dissenting Opinions (FAQ)

<!-- TODO(olshansky): Review & Update -->

**Public facing Fisherman is a vulnerability for censorship and defeats the purpose of decentralized infra.**

Public facing Fishermen are no more subject to censorship than the DAO is. Just like the DAO voters, the more Fishermen participating in the network, the safer the mechanism they control is. The true difference between the DAO voters and Fishermen is the Fishermen’s scope of control is limited only to the quality of service enforcement and the DAO’s control spans the entirety of the system. If the DAO were to ever be compromised by an adversary, the Validators are able to fork the chain and install a new governing body through a social consensus of the ACL. In a parallel situation, if the Fishermen were ever compromised, the DAO would be able to modify the actors without forking the chain and halting service. The architectural separation of powers between Validators, Servicers, Fishermen, and the DAO is what truly ensures the safety of Pocket Network.

**Quality of Service is a second order metric thus it is not a good criteria for salaries of Servicers.**

Quality of Service is harder to measure than Quantity of Service, but it is not a second order metric. Total value of anything is measured as: (How Much X How Pure) minus the cost of making it pure.

**The DAO is not a silver bullet, it’s not trustless, it’s a potential attack vector.**

A powerful government might be able to temporarily stop quality of service metrics but the chain continues. There's no realism in thinking that an overtaking of the DAO isn't a catastrophic event for any version of Pocket Network. As long as there are params like 'MaxValidators' or 'RewardForRelays' the DAO may stop Pocket Network until a social fork happens. Fishermen are the same way. Stop the Fishermen, the DAO will instill new ones. Stop the DAO, the Validators will add a new DAO.

**Publicly identifying fishermen puts them at risk of attacks in the real world.**

Sure, this is a concern that all people in public office deal with.

**Fishermen are not truly Applications, rather a proxy, so the accuracy of their TestScores is questionable and not a valid criteria for salaries of Servicers.**

From a Servicer’s perspective, Fishermen are indistinguishable from applications and therefore are not only valid criteria, but - in fact - preferable sources of information because they are financially incentivised to collect honest and complete reports. Experience with V0 has shown that although applications have the ability to enforce some aspects of Servicer quality, they do not avail themselves of this opportunity.

**The requirements of Fishermen actors are too high. The incentives are oriented properly so POA/POS is overkill.**

If the requirements are too high, the DAO can choose to lower them. Better to start high and lower the bar than to start low and risk flooding the initial system with actors who have little to lose.

**The off-chain monitoring of Fishermen is not secure enough for the duties they are responsible for.**

Off-chain data monitoring and reporting for some aspects of Fishermen activities serves three purposes:

1. It reduces blockchain bloat which is one of the key goals of V1 and an absolute necessity if Pocket network is to grow successfully.
2. It provides a method by which “mad man” actors and Fisherman/Servicer collusion can be detected and punished.
3. Very importantly, it creates a testable, modifiable, tunable environment where large portions of the action and incentive system can be studied without risk of chain state errors. This tested and tuned incentive system is considered a prerequisite for the eventual codification of the system and translation into a fully self-enforced, decentralized solution (Phase II, AKA CastNet)

Keep in mind that good actors by definition do not need to be incentivized to maintain the rules of the system. +2/3 of the network is assumed to want the network to run. Examples of this is how there's no incentive for Validators to propagate Byzantine Evidence in Tendermint and no incentive for Validators to destroy their ephemeral private keys in Algorand's BA. Or even in Hotstuff, there's no incentive for Validators to uphold the integrity of their 'LockedQC' values.

In this case, the people monitoring the Fishermen are the DAO who's largely invested in Network security and integrity and the other network actors who are interacting with the fishermen and are highly incentivized to keep them in-check, like Apps and Servicers. Apps rely on fishermen to enforce the network's decentralized Service Level Agreement. Servicers rely on fishermen to accurately shape their report cards against their peers.

Vitalik on Social Coordination argument:

_'This security assumption, the idea of “getting a block hash from a friend”, may seem unrigorous to many; Bitcoin developers often make the point that if the solution to long-range attacks is some alternative deciding mechanism X, then the security of the blockchain ultimately depends on X, and so the algorithm is in reality no more secure than using X directly - implying that most X, including our social-consensus-driven approach, are insecure._

_However, this logic ignores why consensus algorithms exist in the first place. Consensus is a social process, and human beings are fairly good at engaging in consensus on our own without any help from algorithms; perhaps the best example is the Rai stones, where a tribe in Yap essentially maintained a blockchain recording changes to the ownership of stones (used as a Bitcoin-like zero-intrinsic-value asset) as part of its collective memory. The reason why consensus algorithms are needed is, quite simply, because humans do not have infinite computational power, and prefer to rely on software agents to maintain consensus for us. Software agents are very smart, in the sense that they can maintain consensus on extremely large states with extremely complex rulesets with perfect precision, but they are also very ignorant, in the sense that they have very little social information, and the challenge of consensus algorithms is that of creating an algorithm that requires as little input of social information as possible.'_

**Applications handshaking with Fishermen is more burdensome than V0’s requirements for Applications.**

The only additional requirement of V1 for Applications is to provide the Fisherman a limited AAT/Key combination. V1 also removes the difficult decision of Applications configuring for Quality or Throughput. V1 comes with QOS out of the box so the burden of challenging Servicers is completely eliminated.

**Applications granting an AAT to a Fishermen is completely insecure as they are providing the Fishermen access to their authenticator.**

The Fisherman receives a one-time, one-session token which allows the Fisherman to do only one thing: request relays. The Fishermen have no incentive to overuse that ability. Even a mad-man Fisherman is limited in its ability to cause harm if it wished to.

**By ‘unburdening’ Applications from monitoring QOS you have effectively removed their ability to challenge their servicers. This is a downgrade from V0.**

Applications are and will remain free to report bad service. Unfortunately, even with automated assistance and easy to use SDKs, it is deemed highly likely that applications will continue to act in the future as they have in the past. IE: They don’t spend time and energy fixing our product for us. They just walk away. Self-monitored and Self-enforced QOS is seen as the most viable path forward.

**Only requiring Fishermen to prove they executed the (availability) sampling and not produce on-chain evidence of their data accuracy, latency, and volume claims is an exploitable vulnerability which Fishermen can/will take advantage of.**

Each and every metric which is sampled is collected and verifiable on the off-chain systems. Storing that data on- chain is one of the current V0 problems that V1 seeks to solve. As to the exploitability of this particular aspect… (see.. Madman and Collusion in the attack vectors section) the incentive structure of V1 Fisherman rewards is such that there is no economic gain from this activity and the potential loss is quite significant.

**The optimistic submission of TestScore and Proof transactions degrade the overall security and quality of the network.**

Optimistic Submission:

We imagine that an argument could be put together which tries to demonstrate a security reduction because of optimistic submission. However, since the only aspect of security which is being affected is the amount of payment that Servicers receive, the argument would have to restrict itself to only that single aspect of V1 and (in our opinion, at this time) we do not see such an argument as being valid. However, we look forward to review and discussion of any such argument if and when someone presents it.

**The approximation of MaxRelays through hash collisions is probabilistic and will always have inaccuracies. Thus, it is an unfit method to determine Application usage and TotalAvailableReward inflation.**

Although the first statement (probabilistic=inaccurate) is true, the conclusion (=unfit) does not follow from the premise. V0 is currently probabilistic in selection of Servicers as well as quantity of relays per session. As they apply to node payments, V1 removes the probabilistic components of both node selection and quantity of relays, and replaces them with an amalgamated pool from which they are then distributed in a transparent and protocol enforced manner.

We do see (and are working to mitigate) potential “bad luck” sessions where an application might receive less than an appropriate minimum relay amount. This is not seen as a hard problem to overcome. “Good Luck” sessions will also happen on occasion. This is not seen as a problem at all.

**The name of the actor ‘Fishermen’ has other meanings for other projects and should be changed.**

Traditionally, Fishermen are monitoring actors in other Web3 projects. They allow minimum on-chain information by providing off-chain double-checks on the participants who make ‘claims’. Similar to Validators, though our flavor of Fishermen is different and unique, the core functionality of the actor remains the same.

**Ability of Fishermen to replace servicers mid session makes it necessary that the size of the dispatch list for any given session cannot be fixed.** (Open question)

Newly “recruited” nodes need the freedom to respond to application requests even though they are outside of the calculated list for that session. We may be able to limit dispatch list growth to 2 X the computed size however since the Fisherman cannot replace more than the entire original list. This opens a door for applications to double their effective dispatch list but does not allow abuse of relay usage.

**Memos might be important (open question)**

If people want to keep memos (which are helpful in the case of traceability/auditability), a compromise between the full content of the memos and nothing might be the signature of the text that “would be” on the memo using the private key. Of course, this begs the question of how big is too big, and what’s the average and mean sizes for memos in our current chain.

## References

1. [Pocket Network V0 Whitepaper](https://pocket-network-assets.s3-us-west-2.amazonaws.com/pdfs/Pocket-Network-Whitepaper-v0.3.0.pdf)
2. [Pocket Network Forum](https://forum.pokt.network/)
3. [Pocket Network Constitution](https://github.com/pokt-foundation/governance/blob/master/constitution/constitution.md)
4. [Pocket Network V0 Implementation](https://github.com/pokt-network/pocket-core)
5. [Pocket Network V1 Implementation](https://github.com/pokt-network/pocket)
6. [Tendermint](https://tendermint.com/)
7. [Cosmos](https://v1.cosmos.network/sdk)
8. [Uber's H3](https://h3geo.org)
9. [PostGIS](https://postgis.net/)
10. [Token Bucket Algorithm](https://en.wikipedia.org/wiki/Token_bucket)
11. [Bitcoin](https://en.bitcoin.it/wiki/Difficulty)
12. [Ring Signatures](https://en.wikipedia.org/wiki/Ring_signature)
13. [OAuth](https://oauth.net)
