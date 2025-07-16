# Pocket Network Typescript Client

This directory contains the generated Typescript client for interacting with the Pocket network.

## Installation

```bash
npm install
```

## Usage

The primary entry point for interacting with the Pocket network is the `Pocket` class. It can be initialized as follows:

```typescript
import { Pocket } from "./pocket/pocket.client";

const pk = new Pocket({
    rpcURL: "http://localhost:26657",
    apiURL: "http://localhost:1317",
});
```

### Querying for a balance

```typescript
const address = "pokt1...";
const balance = await pk.cosmos.bank.v1beta1.allBalances({ address });
console.log(balance);
```

### Staking a node

```typescript
import { Pocket } from "./pocket/pocket.client";
import { DirectSecp256k1HdWallet } from "@cosmjs/proto-signing";

const wallet = await DirectSecp256k1HdWallet.fromMnemonic("your mnemonic here");
const [account] = await wallet.getAccounts();

const pk = new Pocket({
    rpcURL: "http://localhost:26657",
    apiURL: "http://localhost:1317",
    signer: wallet,
});

const msg = pk.pocket.supplier.MessageComposer.stakeSupplier({
    creator: account.address,
    stakeAmount: {
        denom: "upokt",
        amount: "1000000",
    },
    services: [
        {
            id: "srv1",
            name: "service 1",
        },
    ],
});

const result = await pk.broadcastTx(account.address, [msg]);
console.log(result);
```

### Unstaking a node

```typescript
import { Pocket } from "./pocket/pocket.client";
import { DirectSecp256k1HdWallet } from "@cosmjs/proto-signing";

const wallet = await DirectSecp256k1HdWallet.fromMnemonic("your mnemonic here");
const [account] = await wallet.getAccounts();

const pk = new Pocket({
    rpcURL: "http://localhost:26657",
    apiURL: "http://localhost:1317",
    signer: wallet,
});

const msg = pk.pocket.supplier.MessageComposer.unstakeSupplier({
    creator: account.address,
});

const result = await pk.broadcastTx(account.address, [msg]);
console.log(result);
```

### Migrating a node

**Note:** The specific details for migration might vary. This is a general example.

```typescript
import { Pocket } from "./pocket/pocket.client";
import { DirectSecp256k1HdWallet } from "@cosmjs/proto-signing";

// It's assumed the node operator has access to the old and new wallets
const oldWallet = await DirectSecp256k1HdWallet.fromMnemonic("old mnemonic here");
const [oldAccount] = await oldWallet.getAccounts();

const newWallet = await DirectSecp256k1HdWallet.fromMnemonic("new mnemonic here");
const [newAccount] = await newWallet.getAccounts();

const pk = new Pocket({
    rpcURL: "http://localhost:26657",
    apiURL: "http://localhost:1317",
    signer: oldWallet, // Sign with the old wallet
});

// First, unstake the supplier from the old address
const unstakeMsg = pk.pocket.supplier.MessageComposer.unstakeSupplier({
    creator: oldAccount.address,
});
const unstakeResult = await pk.broadcastTx(oldAccount.address, [unstakeMsg]);
console.log("Unstake result:", unstakeResult);


// Then, stake the supplier to the new address
// NB: A new client is needed to sign with the new wallet
const pkNew = new Pocket({
    rpcURL: "http://localhost:26657",
    apiURL: "http://localhost:1317",
    signer: newWallet, // Sign with the new wallet
});

const stakeMsg = pkNew.pocket.supplier.MessageComposer.stakeSupplier({
    creator: newAccount.address,
    stakeAmount: {
        denom: "upokt",
        amount: "1000000", // Or the amount that was unstaked
    },
    services: [
        {
            id: "srv1",
            name: "service 1",
        },
    ],
});

const stakeResult = await pkNew.broadcastTx(newAccount.address, [stakeMsg]);
console.log("Stake result:", stakeResult);
```
