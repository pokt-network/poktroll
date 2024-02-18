# Presentation: Slide Overview <!-- omit in toc -->

- [Relay Mining Through Decentralized Gateways](#relay-mining-through-decentralized-gateways)
- [Agenda](#agenda)
  - [Why listen to us?](#why-listen-to-us)
- [RPC](#rpc)
  - [Types of RPC](#types-of-rpc)
  - [Blockchains were designed for writes](#blockchains-were-designed-for-writes)
  - [RPC Nodes were build for reads](#rpc-nodes-were-build-for-reads)
  - [RPC Trilemma](#rpc-trilemma)
- [Relay Mining](#relay-mining)
  - [Relay Mining Intuition](#relay-mining-intuition)
  - [Relay Mining at a Glance](#relay-mining-at-a-glance)
  - [Tree Building](#tree-building)
  - [Claim \& Proof Lifecycle](#claim--proof-lifecycle)
  - [Proof Validation](#proof-validation)
- [Probabilistic Proof](#probabilistic-proof)
  - [Probabilistic Proof - Why?](#probabilistic-proof---why)
  - [Probabilistic Proof - How?](#probabilistic-proof---how)
- [Decentralized Gateways](#decentralized-gateways)
  - [The First Hop Problem](#the-first-hop-problem)
  - [Ring Signatures](#ring-signatures)
  - [Types of Gateways](#types-of-gateways)
- [Future Work](#future-work)
  - [Big Ideas](#big-ideas)
  - [Open Questions](#open-questions)
- [Closing Slide](#closing-slide)

---

TODO:

- Show that Applications stake
- Show the actors involved

## Relay Mining Through Decentralized Gateways

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

## RPC

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

### Blockchains were designed for writes

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

## Relay Mining

- `Title`: Relay Mining
- `Subtitle`: Optimistic Rate Limiting

TODO:

- Add an image of someone mining and looking for gold underground.
- Show Roni's that are golden and those that are not

Speaker Notes:

- `Joke`: Roni is not the one mining, but rather the one being mined for

### Relay Mining Intuition

- `Application` stakes on-chain to put $ in escrow for future payments
- `Supplier` stakes on-chain to put $ in escrow for slashing purposes
- Optimistic (Web3) Rate Limiting
- Incentive to maximize opportunity to do high-quality work
- Disincentive to do any free work

TODO:

- Show a circle containing:
  - All incoming relays
  - Handled relays (based on App Stake)
    --> If exceeded, don't respond
  - Mineable relays (based on difficulty)
    --> If unmet, can't be proven on chain
    --> goes into tree
    ---> Committed to on-chain
  - Proven on-chain
    ---> Single Relay

Speaker Notes:

- Web2 rate limiting techniques:
  - Token Bucket, Leaky Bucket
  - Fixed Window, Sliding Window

### Relay Mining at a Glance

Steps (Left):

1. Tree Building (Find Roni off-chain)
2. Claim & Proof Lifecycle (Store Roni on-chain)
3. Proof Validation (Validate Roni)

Technology (Right):

1. Collision Mining
2. Sparse Merkle Trees
3. Commit & Reveal Schemes
4. Digital Signature Schemes
5. (New) Sparse Merkle Sum Tries
6. (New) Closest Merkle Proof

TODO:

- Check out our relay mining paper at arxiv w/ a link

Speaker Notes:

- `Joke`: We talk about trust in blockchains, so lots of relationship analogies

### Tree Building

TODO:

- Show that Supplier can reject request
- If a Supplier accepts (based on session & stake)
- Visualize:
  - Show Roni the Relay w/ a request
  - Show getting a response
  - Serialize it
  - check it's difficult
- Show that on-chain stake and session determine whether the Supplier should
  be accepting / rejecting the Applications request

Speaker Notes:

### Claim & Proof Lifecycle

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

TODO:

- Show the timeline of the claim and proof lifecycle

### Proof Validation

1. Hash(seed) -> path
2. Proof may not necessarily exist
3. We find the closest proof

TODO:

- Show a single Sparse Merkle Trie
- Add an arrow from a leaf to what it contains (like an magnifying glass)

## Probabilistic Proof

- `Title`: Relay Mining
- `Subtitle`: Optimistic Rate Limiting

### Probabilistic Proof - Why?

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

### Probabilistic Proof - How?

- Show that the issue is scalability
- Show that we only do it sometime
- These are configurable parameters
- The network will self adjust them over time

Speaker Notes:

- Not going to go through the details today,but you can check our papers & documentation

## Decentralized Gateways

- `Title`: Decentralized Gateways
- `Subtitle`: A commitment to delegate trust

### The First Hop Problem

- Options: Threshold signatures? Chain Signature? BLS? ECDSA?
- You always need to trust someone
- We talk about the cost of switching providers
- The cost of switching gateways should be just the same
- The Pocket Network has a standardized API scheme
- All gateways build on these shared standards
  - Standards are "advertised" rather than well defined
  - For example, the Ethereum JSON-RPC API is not "on-chain" but all apps will break if you use themop

TODO:

-

Speaker Notes:

-

### Ring Signatures

- Options: Threshold signatures? Chain Signature? BLS? ECDSA?
- Use the image from the blog post

### Types of Gateways

## Future Work

- `Title`: Future Work
- `Subtitle`: Open Problems & Collaboration Opportunities

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

- Quality of Service: SLA / SLOW
- Incentivization: Free / Paid
- Schema Validation: What's offered?
- Cost, latency, verifiability
- Permissionless free markets
- Other interesting gateway approaches: ar.io, near.io, ENS

### Open Questions

- Data Integrity
  - Light Clients
  - Quorum Consensus
- Zero Knowledge
  - More efficient proofs
  - More efficient security
- KZG commitment to multiple leafs
  - As we expand to websockets
- Privacy
- Data Correctness

Speaker Notes:

- How do we differentiate between an inference on a Llama 70B vs 7B model?

## Closing Slide

- Grow your business with GROVE
- Have access to any service in your Pocket
- Thank you to 1kx for the support in this work
