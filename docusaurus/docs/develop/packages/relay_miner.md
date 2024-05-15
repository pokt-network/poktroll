---
title: RelayMiner
sidebar_position: 3
---

# AppGate Server <!-- omit in toc -->

- [What is RelayMiner?](#what-is-relayminer)
- [Architecture Overview](#architecture-overview)
  - [Alternative implementation](#alternative-implementation)
    - [Shared KVStore](#shared-kvstore)
    - [Shared SessionManager](#shared-sessionmanager)
  - [Lean Client](#lean-client)
  - [Starting the RelayMiner](#starting-the-relayminer)

## What is RelayMiner?

## Architecture Overview

The following diagram captures a high-level overview of the `RelayMiner`'s message flow.

```mermaid
flowchart TB

GW["Gateway"]
PC["PocketNode"]

GW <-- RelayRequest<br>RelayResponse --> RP1
GW <-- RelayRequest<br>RelayResponse --> RP2

SM1 -- Submit Claim --> PC
SM2 -- Submit Claim --> PC

subgraph "Supplier"
  subgraph RM1 ["RelayMiner1"]
    RP1["RelayerProxy"]
    RP1 -- Report<br>Served Relay--> M1
    M1["Miner"]
    M1 -- Mined Relay<br>(difficulty filter) --> SM1
    subgraph SMGR1 ["SessionManager"]
      SM1["Manager"]
      SM1 <-- Update SMT<br>Claim Root --> SMT1
      subgraph SMT1 ["SMT"]
        KV1[("KVStore")]
      end
    end
  end

  subgraph RM2 ["RelayMiner2"]
    RP2["RelayerProxy"]
    RP2 -- Report<br>Served Relay--> M2
    M2["Miner"]
    M2 -- Mined Relay<br>(difficulty filter) --> SM2
    subgraph SMGR2 ["SessionManager"]
      SM2["Manager"]
      SM2 -- Update SMT<br>Claim Root --> SMT2
      subgraph SMT2 ["SMT"]
        KV2[("KVStore")]
      end
    end
  end
end
```

### Alternative implementation
#### Shared KVStore

```mermaid
flowchart TB

GW["Gateway"]

GW <-- RelayRequest<br>RelayResponse --> RP1
GW <-- RelayRequest<br>RelayResponse --> RP2

subgraph "Supplier"
  subgraph RM1 ["RelayMiner1"]
    RP1["RelayerProxy"]
    RP1 -- Report<br>Served Relay--> M1
    M1["Miner"]
    M1 -- Mined Relay<br>(difficulty filter) --> SM1
    subgraph SMGR1 ["SessionManager"]
      SM1["Manager"]
      SM1 <-- Update SMT<br>Claim Root --> SMT1
      SMT1["SMT"]
    end
  end

  subgraph RM2 ["RelayMiner2"]
    RP2["RelayerProxy"]
    RP2 -- Report<br>Served Relay--> M2
    M2["Miner"]
    M2 -- Mined Relay<br>(difficulty filter) --> SM2
    subgraph SMGR2 ["SessionManager"]
      SM2["Manager"]
      SM2 <-- Update SMT<br>Claim Root --> SMT2
      SMT2[SMT]
    end
  end

  SMT1 <-- PUT<br>GET --> KV
  SMT2 <-- PUT<br>GET --> KV

  KV[(KVStore)]
end

SM1 -- Submit Claim --> PC
SM2 -- Submit Claim --> PC

PC["PocketNode"]
```

#### Shared SessionManager

```mermaid
flowchart TB

GW["Gateway"]

GW <-- RelayRequest<br>RelayResponse --> RP1
GW <-- RelayRequest<br>RelayResponse --> RP2

subgraph "Supplier"
  subgraph "RelayMiner1"
    RP1["RelayerProxy"]
    RP1 -- Report<br>Served Relay --> M1
    M1["Miner"]
  end

  subgraph "RelayMiner2"
    RP2["RelayerProxy"]
    RP2 -- Report<br>Served Relay --> M2
    M2["Miner"]
  end

  M1 -- Mined Relay<br>(difficulty filter) --> SM
  M2 -- Mined Relay<br>(difficulty filter) --> SM


  subgraph "SessionManager"
    SM["Manager"]
    SM -- Update SMT<br>Claim Root --> SMT
    subgraph "SMT"
      KV[("KVStore")]
    end
  end
end

SM -- Submit Claim --> PC

PC["PocketNode"]
```

### Lean Client

### Starting the RelayMiner

