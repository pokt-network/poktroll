# Presentation: Slide Overview <!-- omit in toc -->

- [Intro Slide: Relay Mining Through Decentralized Gateways](#intro-slide-relay-mining-through-decentralized-gateways)
- [Agenda](#agenda)
  - [Why listen to us?](#why-listen-to-us)
- [Intro Slide: RPC](#intro-slide-rpc)
  - [Types of RPC](#types-of-rpc)
  - [Blockchains were designed for Writes](#blockchains-were-designed-for-writes)
  - [RPC Nodes were build for reads](#rpc-nodes-were-build-for-reads)
  - [RPC Trilemma](#rpc-trilemma)
- [Intro Slide: Relay Mining](#intro-slide-relay-mining)
  - [Intuition](#intuition)
- [Relay Mining Steps](#relay-mining-steps)
- [Tree Building](#tree-building)
- [Claim \& Proof Lifecycle](#claim--proof-lifecycle)
- [Proof Validation](#proof-validation)
- [Intro Slide: Probabilistic Proof](#intro-slide-probabilistic-proof)
  - [Why is this necessary?](#why-is-this-necessary)
  - [Need to solve for the long tail](#need-to-solve-for-the-long-tail)
- [Intro Slide: Decentralized Gateways](#intro-slide-decentralized-gateways)
  - [Whom do you trust?](#whom-do-you-trust)
  - [Which signatures?](#which-signatures)
- [Intro Slide: Future Work](#intro-slide-future-work)
  - [Who else is doing work in this space?](#who-else-is-doing-work-in-this-space)
  - [Big Ideas](#big-ideas)
  - [Open Questions](#open-questions)
- [Closing Slide](#closing-slide)

---

## Intro Slide: Relay Mining Through Decentralized Gateways

- `Title`: Relay Mining Through Decentralized Gateways
- `Subtitle`: The journey for Roni the Relay

TODO:

- `Joke` Add a screenshot of me last year showing a screenshot of me in the year before
- Show Roni the Relay with a smiley face
- Anthropomorphize Roni the Relay so it sticks and you remember them

## Agenda

1. RPC
2. Relay Mining
3. Probabilistic Proofs
4. Decentralizing Gateways
5. Future Work

TODO:

- Animate in the following two big questions:
  1. How do we incentivize read-only RPC requests?
  2. How do we delegate trust across multiple gateways?

Speaker Notes:

- `Joke`: You might be able to get free lunch here at ETH denver, but there's no free RPC.

### Why listen to us?

- Creators of Pocket Network, powering Grove (Pocket's largest gateway)
- Live MainNet for 3+ years
- 700B+ total relays; 400M+ daily relays; 50+ blockchains
- Actively manage a network of 1,000 validators after overcoming scaling issues of 10,000 validators

TODO:

- Add image from poktscan.com showing the number of nodes & validators

Speaker Notes:

- I'm here on behalf of a lot of people representing both Pocket Network & Grove
- We're doing a full protocol rewrite to build something that'll outlive us
- Will be discussing a combination of things in prod & in progress
- `Joke`: Also, I'm already on stage so you don't really have a choice anyhow

## Intro Slide: RPC

- `Title`: RPC
- `Subtitle`: Remote Procedure Call

TODO:

- Add the graphic that shows client, server, host and port

### Types of RPC

TODO:

- Show Roni the in 3 states:
  1. Carrying data w/ the request
  2. Carrying data w/ the response
  3. Triggering the server to do a lot of work

Speaker Notes:

- Animate a circle around the host and say:
  - This is the thing we're trying to decentralize
  - This is the thing we're trying to incentivize
- A client asks a server to do something
- Why do we want to decentralize it? Because ...

### Blockchains were designed for Writes

- Performant
  - Latency: time-to-finality (block time)
  - Performant: throughput as transactions / second
- Reliable
  - Safety - make the correct progress
  - Liveness - make progress
- Cost
  - Cost of a transaction
  - Cost of on-chain storage
- Payment:
  - Gas / Tx Fees
  - Storage Fees
  - Validator Rewards

TODO:

- Compare Near, Solana, Ethereum, and Aptos

Speaker Notes:

- Blockchains are optimized for `secure state transitions`
- This is how we measure blockchain scalability
- This is how we measure blockchain usage
- This is one of the reasons gas & tx fees are seen as analogs
  to auctions
- BUT, these are all write operations

References:

- https://pontem.medium.com/a-detailed-guide-to-blockchain-speed-tps-vs-80c1d52402d0
- https://shardeum.org/blog/latency-throughput-blockchain/#Latency_in_Blockchain
- https://coincodex.com/article/14198/layer-1-performance-comparing-6-leading-blockchains/
- https://medium.com/nearweek/near-protocol-other-layer-1-solutions-a-comparison-9a91a194dded

### RPC Nodes were build for reads

- Reliable:
  - Availability & uptime
  - Meeting SLAs & SLOs
- Performant:
  - Latency (round trip time)
  - Censorship resistance
- Cost:
  - Per query, per request
  - Per compute unit
  - Per Token
- Payment only through Gateway
  - Reasoning why we got so many providers: https://rpclist.com/
  - Provides: - API Keys - Dashboards - Bells & Whistles - Team Management - Rate Limiting

TODO:

- Show a big circle with write relays (containing Tx)
- Show a small circle showing writes (containing a query)

Speaker Notes:

- But when you're starting to build something, you usually want to read
- Think about how you use the internet
- Most
- `Question`: Who here has ever been hit by rate limiting or large egress fees?

### RPC Trilemma

- Reliability
- Performance
- Cheap (Cost-effective)

TODO:

- Add a diagram of the RPC Trilemma

Speaker Notes:

- Add the triangle
- Talk about how it relates to Web1, Web2, Web3

## Intro Slide: Relay Mining

- `Title`: Relay Mining
- `Subtitle`: Optimistic Rate Limiting

TODO:

- Add an image of someone mining and looking for gold underground.
- Show Roni's that are golden and those that are not

Speaker Notes:

- `Joke`: Roni is not the one mining, but rather the one being mined for

### Intuition

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

## Relay Mining Steps

1. Tree Building (Find Roni)
2. Claim & Proof Lifecycle (Commit to Roni)
3. Proof Validation (Reveal Roni)

Speaker Notes:

- `Joke`: We talk about trust in blockchains, so lots of relationship analogies

## Tree Building

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

## Proof Validation

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

## Intro Slide: Future Work

- `Title`: Future Work
- `Subtitle`: Collaboration Opportunities

Speaker Notes:

- Ideas we discuss
- No immediate timeline
- Would love to work with others
- Reach out

### Who else is doing work in this space?

- near.io
- ar.io
- Permanent domains
- END

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

- Quality of Service: SLA / SLOW
- Incentivization: Free / Paid
- Schema Validation: What's offered?
- Cost, latency, verifiability
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
