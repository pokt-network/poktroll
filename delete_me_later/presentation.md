# Presentation: Slide Overview <!-- omit in toc -->

- [Intro Slide: Relay Mining Through Decentralized Gateways](#intro-slide-relay-mining-through-decentralized-gateways)
- [Agenda](#agenda)
  - [Why should you listen to us?](#why-should-you-listen-to-us)
- [\[WIP\] Intro Slide: RPC](#wip-intro-slide-rpc)
  - [General](#general)
  - [Reads vs Writes](#reads-vs-writes)
  - [Show the RPC Trilemma](#show-the-rpc-trilemma)
  - [Open Problems with RPC](#open-problems-with-rpc)
- [Intro Slide: Relay Mining](#intro-slide-relay-mining)
  - [Analogies](#analogies)
  - [Rate Limiting](#rate-limiting)
- [Claim \& Proof Lifecycle](#claim--proof-lifecycle)
- [Intro Slide: Probabilistic Proof](#intro-slide-probabilistic-proof)
  - [Why is this necessary?](#why-is-this-necessary)
  - [Need to solve for the long tail](#need-to-solve-for-the-long-tail)
- [Intro Slide: Decentralized Gateways](#intro-slide-decentralized-gateways)
  - [Whom do you trust?](#whom-do-you-trust)
  - [Which signatures?](#which-signatures)
  - [Who else is doing work in this space?](#who-else-is-doing-work-in-this-space)
- [Intro Slide: Future Work](#intro-slide-future-work)
  - [Big Ideas](#big-ideas)
  - [Open Questions](#open-questions)
- [Closing Slide](#closing-slide)

---

## Intro Slide: Relay Mining Through Decentralized Gateways

- `Title`: Relay Mining Through Decentralized Gateways
- `Subtitle`: The journey for Roni the Relay

TODO:

- `Joke` Add a screenshot of me last year showing a screenshot of me in the year before

## Agenda

1. RPC
2. Relay Mining
3. Probabilistic Proofs
4. Decentralizing Gateways
5. Future Work

TODO:

- Animate in the following two big questions:
  1. How do we incentivize read RPC requests from full nodes?
  2. How do we delegate and manage trust across multiple gateways?

Speaker Notes:

- There's no free RPC, and I will be discussing the tradeoffs we need to make

### Why should you listen to us?

- Pocket Network is the largest decentralized RPC network
- Managing a validator set of 1,000 after hitting scaling issues at 10,000
- Grove is the primary gateway that provides access to Pocket Network today
- 3+ years on MainNet
- 50+ blockchains
- 400M+ daily relays
- 700B+ total relays

Speaker Notes:

- Shannon (in progress) is a rewrite of Morse (in production)
- Will be discussing a combination of things in prod & in progress
- `Joke`: I mentioned that we'll be discussing Roni, but in reality they have lots of cousins

## [WIP] Intro Slide: RPC

- `Title`: RPC
- `Subtitle`: Remote Procede Call

- Show how big RPC is
- Can I find stats on this specifically?

TODO:

- Add the animation that shows host and port

Speaker Notes:

### General

- https://rpclist.com/
- The start to every developer's journey
- We no longer run our own nodes
- This is how development starts
- Need an endpoint
- Need a host & port
- Show what an RPC is
- What questions to developers actually start by asking:
- How do I read data from a blockchain?
- How do I write data to a blockchain?
- How can I get an RPC node?
- How much does it cost
- What is the quality of his?

### Reads vs Writes

- Blockchains are optimized for `secure state transitions`
- Blockchain scalability / usage is measured in writes
  - EigenLayer: uses throughput
  - L1s: use tx/s
  - Ethereum: block times & gas fees
- Actual blockchain usage can also be measured by the amount of reads
- Block storage
- Auctions
- Paying for block storage
-

### Show the RPC Trilemma

- Reliability
- Performance
- Cost-effectiveness
- Add the triangle
- Talk about how it relates to Web1, Web2, Web3

### Open Problems with RPC

- Quality of Service: SLA / SLOW
- Incentivization: Free / Paid
- Schema Validation: What's offered?
- Cost, latency, verifiability

## Intro Slide: Relay Mining

- `Title`: Relay Mining
- `Subtitle`: Optimistic rate limiting

TODO:

- Add an image of someone mining and looking for gold underground.
- Show Roni's that are golden and those that are not

Speaker Notes:

- Different
- `Joke`: Roni is not the one mining, but rather the one being mined for

### Analogies

- Big Circle: all relays incoming
- Smaller circle: all relays handled (based on stake)
- Smaller circle: all relays that can be mined (based on difficulty)
- Smaller circle: all relays that go on chain
- Small circle with one relay: the relay that needs to be provend

- Web2: Count and Rate Limit
- Serve relays for free until you get paid for some of them
-
- Compare it to Bitcoin & PoW
- Compare it to Arweave & Proof of Access
- Compare it to Chia & Proof of Space
- `Question`: How do we incentivize doing work for read access?
- `Questions`: How do we incentivize to provide high quality on both reads & writes?

Speaker Notes:

- We're tackling problems with RPC from the user's POV
  - Cost, QoS, Reliability, etc...
- To do those, the other sides needs an incentive
- Rate Limiting is a key part of this, and is at the foundation of this economic model
- Web2: Rate Limiting
- Web3: Relay Mining:

### Rate Limiting

- How do we do rate limiting in web2?
- Charge per token (AIs)
- Charge per request (anyone)
- Charge for bandwidth (egress/increases)
- Rate Limiting Algorithms
  - Window
  - Bucketing
  - Trust
  - `Question`: How many people here challenged what an API endpoint said your usage actually is?How many

## Claim & Proof Lifecycle

- Show the relay mining process
  - Show the Claim & Proof Lifecycle
  - Show the tree
    - Show a tree
    - Show a branch
    - Show the claim and proof lifecycle
    - Show that we prove a single branch
- References
  - Reference Ramiro's work in the paper for detailed analysis
  - Reference the other whitepaper I was given for KZG commitments

## Intro Slide: Probabilistic Proof

### Why is this necessary?

- Out of necessity to scale
- Out of necessity to make things permissionless
- Out of necessary to make things decentralized
- Want to enable:
  - Permissionless Gateways
  - Permissionless Services
  - Permissionless Applications
- There will be:
  - Spam on the network
  - Bloat on the network
  - Self-dealing attacks on the network

### Need to solve for the long tail

- Show that the issue is scalability
- Show that we only do it sometime
- These are configurable parameters
- The network will self adjust them over time

## Intro Slide: Decentralized Gateways

- Options: Threshold signatures? Chain Signature? BLS? ECDSA?

### Whom do you trust?

- You always need to trust someone
- We talk about the cost of switching providers
- The cost of switching gateways should be just the same
- The Pocket Network has a standardized API scheme
- All gateways build on these shared standards
  - Standards are "advertised" rather than well defined
  - For example, the Ethereum JSON-RPC API is not "on-chain" but all apps will break if you use themop

### Which signatures?

- Options: Threshold signatures? Chain Signature? BLS? ECDSA?
- Use the image from the blog post

### Who else is doing work in this space?

- near.io
- ar.io
- Permanent domains
- END

## Intro Slide: Future Work

- `Title`: Future Work
- `Subtitle`: Collaboration Opportunities

Speaker Notes:

- Ideas we discuss
- No immediate timeline
- Would love to work with others
- Reach out

### Big Ideas

- Quality Incentive Protocol:
  - On-Chain QoS
  - Incentivized TCP/IP
- Permissionless Gatewas
- Permissionless Services
- AI Gateways
- Value added by gateways
- Marketplace of Dynamic Pricing
- Compute Units
- Optimistic rate limiting -> multi-tenant rate limiting
  - Can GCP & AWS charge a single account?
- Permissionless free markets

### Open Questions

- Data Integrity
  - Light Clients
  - Quorum Consensus
- Zero Knowledge
  - More efficient proofs
  - More efficient security
- KZG commitment to multiple leafs
  - As we expand to websockets

## Closing Slide

- Grow your business with GROVE
- Have access to any service in your Pocket
- Thank you to 1kx for the support in this work
