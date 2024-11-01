---
title: RPC Endpoints
sidebar_position: 3
---

## TestNet RPC Endpoints

We have provided `gRPC`, `JSON-RPC` and `REST` endpoints, which are available here:

- `https://shannon-testnet-grove-seed-rpc.poktroll.com`
- `https://shannon-testnet-grove-seed-grpc.poktroll.com`
- `https://shannon-testnet-grove-seed-api.poktroll.com`

You can review the difference between them in the [Cosmos SDK docs](https://docs.cosmos.network/main/learn/advanced/grpc_rest#comparison-table).

### JSON-RPC

Using `curl`:

```bash
curl -X POST https://shannon-testnet-grove-seed-rpc.poktroll.com/block
```

Using the `poktrolld` binary:

```bash
poktrolld query block --type=height 0 --node https://shannon-testnet-grove-seed-rpc.poktroll.com
```

### gRPC

_TODO_TECHDEBT: Add a gRPC example_

### REST

_TODO_TECHDEBT: Add a REST example_
