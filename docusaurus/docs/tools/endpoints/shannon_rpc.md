---
title: Shannon RPC Endpoints
sidebar_position: 1
---

- [Types of RPC Endpoints](#types-of-rpc-endpoints)
- [Beta TestNet](#beta-testnet)
  - [Beta RPC Endpoints](#beta-rpc-endpoints)
  - [Beta JSON-RPC Example](#beta-json-rpc-example)
- [Alpha TestNet](#alpha-testnet)
  - [Alpha RPC Endpoints](#alpha-rpc-endpoints)
  - [Alpha JSON-RPC Example](#alpha-json-rpc-example)
- [Genesis](#genesis)

## Types of RPC Endpoints

You can review the difference between them in the [Cosmos SDK docs](https://docs.cosmos.network/main/learn/advanced/grpc_rest#comparison-table).

## Beta TestNet

### Beta RPC Endpoints

We provide `gRPC`, `JSON-RPC` and `REST` endpoints, which are available here:

- **RPC**: `https://shannon-testnet-grove-rpc.beta.pocket.com`
- **gRPC**: `https://shannon-testnet-grove-grpc.beta.pocket.com`
- **REST**: `https://shannon-testnet-grove-api.beta.pocket.com`

### Beta JSON-RPC Example

Using `curl`:

```bash
curl -X POST https://shannon-testnet-grove-seed-rpc.pocket.com/block
```

Using the `pocketd` binary:

```bash
pocketd query block --type=height 0 --node https://shannon-testnet-grove-seed-rpc.pocket.com
```

## Alpha TestNet

### Alpha RPC Endpoints

We provide `gRPC`, `JSON-RPC` and `REST` endpoints, which are available here:

- **RPC**: `https://shannon-testnet-grove-rpc.alpha.pocket.com`
- **gRPC**: `https://shannon-testnet-grove-grpc.alpha.pocket.com`
- **REST**: `https://shannon-testnet-grove-api.alpha.pocket.com`

### Alpha JSON-RPC Example

Using `curl`:

```bash
curl -X POST https://shannon-testnet-grove-seed-rpc.alpha.pocket.com/block
```

Using the `pocketd` binary:

```bash
pocketd query block --type=height 1 --node https://shannon-testnet-grove-seed-rpc.alpha.pocket.com
```

## Genesis

The genesis file for the Pocket Network is located at [pokt-network-genesis](https://github.com/pokt-network/pocket-network-genesis).
