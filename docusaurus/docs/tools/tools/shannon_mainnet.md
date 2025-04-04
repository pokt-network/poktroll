---
title: MainNet
sidebar_position: 1
---

- 🗺️ [TODO: StakeNode's Explorer](http://todo.com)
- 🗺️ [Soothe's Explorer](https://shannon-mainnet.trustsoothe.io)
- 👨‍💻 [Soothe's GraphQL Playground](https://shannon-mainnet-api.trustsoothe.io)

## RPC Endpoints

We provide `gRPC`, `JSON-RPC` and `REST` endpoints, which are available here:

- **RPC**: `https://shannon-grove-rpc.mainnet.poktroll.com`
- **gRPC**: `https://shannon-grove-grpc.mainnet.poktroll.com`
- **REST**: `https://shannon-grove-api.mainnet.poktroll.com`

### MainNet JSON-RPC Example

Using `curl`:

```bash
curl -X POST https://shannon-grove-rpc.mainnet.poktroll.com/block
```

Using the `pocketd` binary:

```bash
pocketd query block --type=height 69 --node=https://shannon-grove-rpc.mainnet.poktroll.com
```

## MainNet Genesis

The genesis file for the Pocket Network is located at [pokt-network-genesis](https://github.com/pokt-network/pocket-network-genesis/tree/master/shannon/mainnet).
