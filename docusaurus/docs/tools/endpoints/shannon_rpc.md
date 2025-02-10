---
title: Shannon RPC Endpoints
sidebar_position: 3
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

- **RPC**: `https://shannon-testnet-grove-rpc.beta.poktroll.com`
- **gRPC**: `https://shannon-testnet-grove-grpc.beta.poktroll.com`
- **REST**: `https://shannon-testnet-grove-api.beta.poktroll.com`

### Beta JSON-RPC Example

Using `curl`:

```bash
curl -X POST https://shannon-testnet-grove-seed-rpc.poktroll.com/block
```

Using the `poktrolld` binary:

```bash
poktrolld query block --type=height 0 --node https://shannon-testnet-grove-seed-rpc.poktroll.com
```

## Alpha TestNet

### Alpha RPC Endpoints

We provide `gRPC`, `JSON-RPC` and `REST` endpoints, which are available here:

- **RPC**: `https://shannon-testnet-grove-rpc.alpha.poktroll.com`
- **gRPC**: `https://shannon-testnet-grove-grpc.alpha.poktroll.com`
- **REST**: `https://shannon-testnet-grove-api.alpha.poktroll.com`

### Alpha JSON-RPC Example

Using `curl`:

```bash
curl -X POST https://shannon-testnet-grove-seed-rpc.alpha.poktroll.com/block
```

Using the `poktrolld` binary:

```bash
poktrolld query block --type=height 1 --node https://shannon-testnet-grove-seed-rpc.alpha.poktroll.com
```

## Genesis

The genesis file for the Pocket Network is located at [pokt-network-genesis](https://github.com/pokt-network/pocket-network-genesis).
