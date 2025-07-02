# Module Architecture and Relationships

## Core Blockchain Modules (`/x/`)

### Primary Economic Modules
- **application** - App staking and delegation for API access
  - Location: `x/application/`
  - Key files: `keeper/application.go`, `types/application.go`
  - Handles app staking, gateway delegation, unstaking lifecycle

- **supplier** - Service provider (RelayMiner) management  
  - Location: `x/supplier/`
  - Key files: `keeper/supplier.go`, `types/supplier.go`
  - Manages supplier registration, staking, and service provisioning

- **gateway** - Quality-of-service layer for enterprise usage
  - Location: `x/gateway/`
  - Handles enterprise QoS, app-to-supplier routing

### Protocol Coordination Modules
- **service** - API service registry and relay mining difficulty
  - Location: `x/service/`
  - Manages service definitions, relay mining difficulty adjustment
  - Key files: `x/service/keeper/`, `proto/pocket/service/`

- **session** - Time-bounded interaction windows between apps/suppliers
  - Location: `x/session/`
  - Coordinates session lifecycle, app-supplier pairing
  - Critical for relay request routing

### Verification & Settlement Modules  
- **proof** - Cryptographic verification of API usage for settlements
  - Location: `x/proof/`
  - Handles relay proof submission and verification
  - Works with `pkg/crypto/rings/` for privacy-preserving authentication

- **tokenomics** - Economic incentives, penalties, and token distribution
  - Location: `x/tokenomics/`  
  - Settlement calculations, reward distribution, penalty enforcement
  - Integrates with all other modules for economic interactions

### Utility Modules
- **shared** - Cross-module utilities and constants
  - Location: `x/shared/`
  - Common types, constants, and utility functions
  - Used by multiple modules to avoid circular dependencies

- **migration** - Protocol upgrade and data migration utilities
  - Location: `x/migration/`
  - Handles Morse-to-Shannon upgrade logic
  - Key files: `keeper/morse_claimable_account.go`

## Key Component Relationships

### Relay Request Flow
1. **Application** stakes tokens and optionally delegates to **Gateway**
2. **Session** module creates time-bounded session between app and **Supplier**
3. **Service** module defines available APIs and mining difficulty
4. **Supplier** serves API requests and generates **Proof**
5. **Tokenomics** settles payments and distributes rewards

### Module Dependencies
- **Session** ← depends on → **Application**, **Supplier**, **Service**
- **Proof** ← depends on → **Session**, **Supplier** 
- **Tokenomics** ← depends on → **Proof**, **Session**, **Application**, **Supplier**
- **Gateway** ← depends on → **Application**
- **Shared** ← used by → All modules

## Directory Structure Patterns

### Per-Module Organization
```
x/<module>/
├── keeper/          # State management, business logic
├── module/          # Cosmos SDK module integration  
├── simulation/      # Simulation testing
└── types/          # Protocol buffer types, messages, validation
```

### Key Keeper Files
- `keeper.go` - Main keeper struct and initialization
- `msg_server.go` - Transaction message handling
- `query.go` - Query handler implementations
- `<entity>.go` - Core business logic (e.g., `application.go`)

### Generated Protocol Files
- `api/pocket/<module>/` - Generated from `proto/pocket/<module>/`
- Includes `.pulsar.go` files for state and message types
- Updated via `make proto_regen`