# IBC Integration Guide

Inter-Blockchain Communication (IBC) enables secure and reliable communication between independent blockchains. Poktroll leverages IBC to connect with the broader Cosmos ecosystem, allowing seamless asset transfers and cross-chain interactions.

## Quick Start

Choose your path based on your needs:

### 🚀 For Integrators
Connect your blockchain to Poktroll:
- [Chain Integration Guide](./integration.md) - Connect external chains to Poktroll

### 🛠️ For Developers  
Build and test with IBC:
- [LocalNet Development](./localnet.md) - Local development and testing
- [Testing & Debugging](./testing.md) - Comprehensive testing strategies

### 🔧 For Advanced Users
Explore advanced IBC features:
- [Packet Forward Middleware (PFM)](./pfm.md) - Multi-hop transfers
- [Interchain Accounts (ICA)](./ica.md) - Cross-chain account control

## What is IBC?

IBC is a standardized protocol that enables different blockchains to communicate and transfer data securely. Think of it as the "internet of blockchains" - it allows independent networks to:

- **Transfer tokens** between chains without centralized exchanges
- **Share data** and trigger actions across multiple blockchains  
- **Maintain sovereignty** while enabling interoperability

## IBC in Poktroll

Poktroll uses IBC to:

1. **Connect to Cosmos Hub** - Access the broader Cosmos ecosystem
2. **Enable cross-chain payments** - Accept payments from any IBC-enabled chain
3. **Facilitate governance** - Allow cross-chain governance participation
4. **Support multi-chain APIs** - Serve applications across different blockchains

## Architecture Overview

```
┌─────────────┐    IBC     ┌─────────────┐    IBC     ┌─────────────┐
│   Chain A   │◄──────────►│  Poktroll   │◄──────────►│   Chain B   │
│             │   Channel  │             │   Channel  │             │
└─────────────┘            └─────────────┘            └─────────────┘
       │                          │                          │
       └─────────────── Relayer Network ────────────────────┘
```

### Key Components

- **Channels**: Bidirectional communication pathways between chains
- **Relayers**: Off-chain processes that relay packets between chains
- **Light Clients**: Verify the state of remote chains
- **Packet Forward Middleware**: Enable multi-hop transfers

## Getting Started

### Prerequisites

Before integrating with Poktroll via IBC, ensure you have:

- ✅ An IBC-enabled Cosmos SDK blockchain
- ✅ Reliable RPC/REST infrastructure  
- ✅ Access to a relayer service
- ✅ Understanding of your chain's governance process

### Connection Process

1. **Establish Connection** - Create IBC connection between chains
2. **Open Channel** - Set up communication channel for specific applications
3. **Configure Relayer** - Deploy or connect to relayer infrastructure
4. **Test Transfers** - Verify bi-directional token transfers
5. **Monitor & Maintain** - Ongoing relayer and channel health monitoring

### Resources

- [Cosmos IBC Documentation](https://ibc.cosmos.network/)
- [IBC Protocol Specification](https://github.com/cosmos/ibc)
- [Hermes Relayer Guide](https://hermes.informal.systems/)

## IBC Core Concepts

### IBC Protocol Layers

* **IBC/TAO (Transport, Authentication, Ordering)** – the cross‑chain infrastructure handling packet lifecycle.
* **IBC/App** – the application layer, including modules like ICS‑20 (fungible token transfers), ICS‑721 (NFTs), ICS‑27 (Interchain Accounts), and others.

### Core Components:

1. [**Client**](https://tutorials.cosmos.network/academy/3-ibc/4-clients.html) – each chain maintains a light client of the other.
2. [**Connection**](https://tutorials.cosmos.network/academy/3-ibc/2-connections.html) – four‑step handshake (`ConnOpenInit/Try/Ack/Confirm`).
3. [**Channel**](https://tutorials.cosmos.network/academy/3-ibc/3-channels.html) – attached to a connection, providing ordered or unordered packet transport.
4. [**Relayer**](https://tutorials.cosmos.network/academy/2-cosmos-concepts/13-relayer-intro.html) – off‑chain service that listens for IBC events and relays packet proof messages between chains (e.g. [Hermes](https://hermes.informal.systems/)).

---

**Next Steps**: Choose a guide from the sections above to begin your IBC integration journey with Poktroll.