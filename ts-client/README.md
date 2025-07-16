# Pocket Network TypeScript Client <!-- omit in toc -->

A comprehensive TypeScript client for interacting with the Pocket Network Shannon upgrade blockchain.

## Table of Contents <!-- omit in toc -->

- [Installation](#installation)
- [Running the Examples](#running-the-examples)
  - [ES Module Support](#es-module-support)
  - [Alternative: .mjs Extension](#alternative-mjs-extension)
- [Live Network Examples](#live-network-examples)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Querying the Network](#querying-the-network)
  - [Account Balances](#account-balances)
  - [Supplier Information](#supplier-information)
  - [Application Information](#application-information)
  - [Gateway Information](#gateway-information)
  - [Service Information](#service-information)
  - [Session Information](#session-information)
  - [Proof and Claims](#proof-and-claims)
- [Staking Operations](#staking-operations)
  - [Staking a Supplier](#staking-a-supplier)
  - [Unstaking a Supplier](#unstaking-a-supplier)
  - [Staking an Application](#staking-an-application)
  - [Unstaking an Application](#unstaking-an-application)
  - [Staking a Gateway](#staking-a-gateway)
  - [Unstaking a Gateway](#unstaking-a-gateway)
- [Advanced Operations](#advanced-operations)
  - [Application Delegation](#application-delegation)
  - [Service Management](#service-management)
  - [Proof System](#proof-system)
- [Error Handling](#error-handling)

## Installation

```bash
# Using npm
npm install

# Or using yarn (recommended if npm hangs)
yarn install
```

## Running the Examples

To run the examples, you can use the provided quickstart script:

```bash
# Run the quickstart example
npm run quickstart

# Or directly with node
node quickstart.js
```

### ES Module Support

This client uses ES modules. Your `package.json` must include `"type": "module"` and imports must use `.js` extensions:

```json
{
  "type": "module"
}
```

### Alternative: .mjs Extension

If you prefer not to modify package.json, you can use `.mjs` extension for your files:

```bash
# Rename your file to .mjs
mv myfile.js myfile.mjs
node myfile.mjs
```

### Troubleshooting

If you encounter issues with the complex generated TypeScript modules, you can use a simplified approach:

```javascript
// Simple REST API client approach
class PocketClient {
  constructor({ apiURL }) {
    this.apiURL = apiURL;
  }
  
  async query(endpoint) {
    const response = await fetch(`${this.apiURL}${endpoint}`);
    return await response.json();
  }
  
  async getNodeInfo() {
    return await this.query('/cosmos/base/tendermint/v1beta1/node_info');
  }
  
  async getBalance(address, denom = 'upokt') {
    return await this.query(`/cosmos/bank/v1beta1/balances/${address}/${denom}`);
  }
}

// Usage
const client = new PocketClient({
  apiURL: 'https://shannon-grove-api.mainnet.poktroll.com'
});
```

## Live Network Examples

Here are some practical examples you can run against the Shannon Grove mainnet:

```typescript
import { Client } from "./index.js";

// Query network parameters
const client = new Client({
  rpcURL: "https://shannon-grove-rpc.mainnet.poktroll.com",
  apiURL: "https://shannon-grove-api.mainnet.poktroll.com",
});

// Get current network status
const status = await client.cosmos.base.tendermint.v1beta1.getNodeInfo();
console.log("Network status:", status);

// Get latest block
const latestBlock =
  await client.cosmos.base.tendermint.v1beta1.getLatestBlock();
console.log("Latest block:", latestBlock);

// Get all active suppliers
const activeSuppliers = await client.pocket.supplier.queryAllSuppliers();
console.log(`Found ${activeSuppliers.supplier.length} active suppliers`);

// Get all available services
const availableServices = await client.pocket.service.queryAllServices();
console.log(`Found ${availableServices.service.length} available services`);
```

## Quick Start

The client uses the `Client` class for all network interactions. Here's how to get started:

```typescript
import { Client } from "./index.js";

// For queries only (no wallet needed)
const client = new Client({
  rpcURL: "https://shannon-grove-rpc.mainnet.poktroll.com",
  apiURL: "https://shannon-grove-api.mainnet.poktroll.com",
});

// For transactions (wallet required)
import { DirectSecp256k1HdWallet } from "@cosmjs/proto-signing";

const wallet = await DirectSecp256k1HdWallet.fromMnemonic("your mnemonic here");
const clientWithSigner = new Client({
  rpcURL: "https://shannon-grove-rpc.mainnet.poktroll.com",
  apiURL: "https://shannon-grove-api.mainnet.poktroll.com",
  signer: wallet,
});
```

## Configuration

The client supports both mainnet and testnet configurations:

```typescript
// Mainnet (Shannon Grove)
const mainnetClient = new Client({
  rpcURL: "https://shannon-grove-rpc.mainnet.poktroll.com",
  apiURL: "https://shannon-grove-api.mainnet.poktroll.com",
});

// Local development
const localClient = new Client({
  rpcURL: "http://localhost:26657",
  apiURL: "http://localhost:1317",
});
```

## Querying the Network

### Account Balances

```typescript
// Query all balances for an address
const address = "pokt1...";
const balances = await client.cosmos.bank.v1beta1.allBalances({ address });
console.log("Balances:", balances);

// Query specific denomination balance
const balance = await client.cosmos.bank.v1beta1.balance({
  address,
  denom: "upokt",
});
console.log("POKT Balance:", balance);
```

### Supplier Information

```typescript
// Get all suppliers
const suppliers = await client.pocket.supplier.queryAllSuppliers();
console.log("All suppliers:", suppliers);

// Get specific supplier
const supplierAddress = "pokt1...";
const supplier = await client.pocket.supplier.querySupplier({
  operator_address: supplierAddress,
});
console.log("Supplier details:", supplier);

// Get supplier module parameters
const supplierParams = await client.pocket.supplier.queryParams();
console.log("Supplier parameters:", supplierParams);
```

### Application Information

```typescript
// Get all applications
const applications = await client.pocket.application.queryAllApplications();
console.log("All applications:", applications);

// Get specific application
const appAddress = "pokt1...";
const application = await client.pocket.application.queryApplication({
  address: appAddress,
});
console.log("Application details:", application);

// Get application module parameters
const appParams = await client.pocket.application.queryParams();
console.log("Application parameters:", appParams);
```

### Gateway Information

```typescript
// Get all gateways
const gateways = await client.pocket.gateway.queryAllGateways();
console.log("All gateways:", gateways);

// Get specific gateway
const gatewayAddress = "pokt1...";
const gateway = await client.pocket.gateway.queryGateway({
  address: gatewayAddress,
});
console.log("Gateway details:", gateway);
```

### Service Information

```typescript
// Get all services
const services = await client.pocket.service.queryAllServices();
console.log("All services:", services);

// Get specific service
const serviceId = "ethereum-mainnet";
const service = await client.pocket.service.queryService({
  id: serviceId,
});
console.log("Service details:", service);

// Get relay mining difficulty
const difficulty = await client.pocket.service.queryRelayMiningDifficulty({
  service_id: serviceId,
});
console.log("Mining difficulty:", difficulty);
```

### Session Information

```typescript
// Get session information
const sessionInfo = await client.pocket.session.queryGetSession({
  application_address: "pokt1...",
  service_id: "ethereum-mainnet",
  block_height: 1000,
});
console.log("Session info:", sessionInfo);
```

### Proof and Claims

```typescript
// Get all claims
const claims = await client.pocket.proof.queryAllClaims();
console.log("All claims:", claims);

// Get specific claim
const claim = await client.pocket.proof.queryGetClaim({
  session_id: "session_123",
  supplier_operator_address: "pokt1...",
});
console.log("Claim details:", claim);

// Get all proofs
const proofs = await client.pocket.proof.queryAllProofs();
console.log("All proofs:", proofs);
```

## Staking Operations

### Staking a Supplier

```typescript
import { Client } from "./index.js";
import { DirectSecp256k1HdWallet } from "@cosmjs/proto-signing";

const wallet = await DirectSecp256k1HdWallet.fromMnemonic("your mnemonic here");
const [account] = await wallet.getAccounts();

const client = new Client({
  rpcURL: "https://shannon-grove-rpc.mainnet.poktroll.com",
  apiURL: "https://shannon-grove-api.mainnet.poktroll.com",
  signer: wallet,
});

const stakeAmount = {
  denom: "upokt",
  amount: "1000000000", // 1000 POKT (1 POKT = 1,000,000 upokt)
};

const services = [
  {
    id: "ethereum-mainnet",
    name: "Ethereum Mainnet",
    compute_units_per_relay: 1,
    owner_address: account.address,
  },
];

const result = await client.pocket.supplier.sendMsgStakeSupplier(
  account.address,
  {
    signer: account.address,
    stake: stakeAmount,
    services: services,
  }
);

console.log("Supplier staking result:", result);
```

### Unstaking a Supplier

```typescript
const result = await client.pocket.supplier.sendMsgUnstakeSupplier(
  account.address,
  {
    signer: account.address,
  }
);

console.log("Supplier unstaking result:", result);
```

### Staking an Application

```typescript
const stakeAmount = {
  denom: "upokt",
  amount: "1000000000", // 1000 POKT
};

const serviceConfigs = [
  {
    service_id: "ethereum-mainnet",
    compute_units_per_relay: 1,
  },
];

const result = await client.pocket.application.sendMsgStakeApplication(
  account.address,
  {
    address: account.address,
    stake: stakeAmount,
    service_configs: serviceConfigs,
  }
);

console.log("Application staking result:", result);
```

### Unstaking an Application

```typescript
const result = await client.pocket.application.sendMsgUnstakeApplication(
  account.address,
  {
    address: account.address,
  }
);

console.log("Application unstaking result:", result);
```

### Staking a Gateway

```typescript
const stakeAmount = {
  denom: "upokt",
  amount: "1000000000", // 1000 POKT
};

const result = await client.pocket.gateway.sendMsgStakeGateway(
  account.address,
  {
    address: account.address,
    stake: stakeAmount,
  }
);

console.log("Gateway staking result:", result);
```

### Unstaking a Gateway

```typescript
const result = await client.pocket.gateway.sendMsgUnstakeGateway(
  account.address,
  {
    address: account.address,
  }
);

console.log("Gateway unstaking result:", result);
```

## Advanced Operations

### Application Delegation

```typescript
// Delegate application to a gateway
const gatewayAddress = "pokt1...";
const result = await client.pocket.application.sendMsgDelegateToGateway(
  account.address,
  {
    app_address: account.address,
    gateway_address: gatewayAddress,
  }
);

console.log("Delegation result:", result);

// Undelegate from gateway
const undelegateResult =
  await client.pocket.application.sendMsgUndelegateFromGateway(
    account.address,
    {
      app_address: account.address,
      gateway_address: gatewayAddress,
    }
  );

console.log("Undelegation result:", undelegateResult);
```

### Service Management

```typescript
// Add a new service (requires governance)
const newService = {
  id: "my-custom-service",
  name: "My Custom Service",
  compute_units_per_relay: 1,
  owner_address: account.address,
};

const result = await client.pocket.service.sendMsgAddService(account.address, {
  service: newService,
});

console.log("Service addition result:", result);
```

### Proof System

```typescript
// Create a claim
const claim = {
  session_header: {
    application_address: "pokt1...",
    service_id: "ethereum-mainnet",
    session_id: "session_123",
    session_start_block_height: 1000,
    session_end_block_height: 1100,
  },
  root_hash: "0x...", // Merkle root of relays
};

const claimResult = await client.pocket.proof.sendMsgCreateClaim(
  account.address,
  {
    supplier_operator_address: account.address,
    session_header: claim.session_header,
    root_hash: claim.root_hash,
  }
);

console.log("Claim creation result:", claimResult);

// Submit proof
const proof = {
  session_header: claim.session_header,
  closest_merkle_proof: [], // Merkle proof bytes
};

const proofResult = await client.pocket.proof.sendMsgSubmitProof(
  account.address,
  {
    supplier_operator_address: account.address,
    session_header: proof.session_header,
    proof: proof.closest_merkle_proof,
  }
);

console.log("Proof submission result:", proofResult);
```

## Error Handling

Always wrap your operations in try-catch blocks:

```typescript
try {
  const result = await client.pocket.supplier.querySupplier({
    operator_address: "pokt1...",
  });
  console.log("Success:", result);
} catch (error) {
  console.error("Error querying supplier:", error);

  // Handle specific error types
  if (error.code === 5) {
    console.log("Supplier not found");
  }
}
```

**Note**: For transaction operations (staking, unstaking, etc.), you'll need a wallet with POKT tokens. The minimum stake amounts are defined in the network parameters and can be queried using the parameter endpoints shown above.
