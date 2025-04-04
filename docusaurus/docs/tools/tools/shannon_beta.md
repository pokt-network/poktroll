---
title: Beta TestNet
sidebar_position: 2
---

## Explorers

- 🪙 [StakeNode's Faucet](https://faucet.beta.testnet.pokt.network/)
- 🗺️ [StakeNode's Explorer](https://shannon.beta.testnet.pokt.network)
- 🗺️ [Soothe's Explorer](https://shannon-beta.trustsoothe.io)
- 👨‍💻 [Soothe's GraphQL Playground](https://shannon-beta-api.trustsoothe.io/)

## RPC Endpoints

We provide `gRPC`, `JSON-RPC` and `REST` endpoints, which are available here:

- **RPC**: `https://shannon-testnet-grove-rpc.beta.poktroll.com`
- **gRPC**: `https://shannon-testnet-grove-grpc.beta.poktroll.com`
- **REST**: `https://shannon-testnet-grove-api.beta.poktroll.com`

### Beta JSON-RPC Example

Using `curl`:

```bash
curl -X POST https://shannon-testnet-grove-seed-rpc.poktroll.com/block
```

Using the `pocketd` binary:

```bash
pocketd query block --type=height 0 --node https://shannon-testnet-grove-seed-rpc.poktroll.com
```

## Alpha Genesis

The genesis file for the Pocket Network is located at [pokt-network-genesis](https://github.com/pokt-network/pocket-network-genesis/tree/master/shannon/testnet-beta).
