---
title: Alpha TestNet
sidebar_position: 3
---

## Explorers

- 🚰 Coming Soon: MACT Faucet for claiming tokens
- 🪙 [StakeNodes' Faucet](https://faucet.alpha.testnet.pokt.network/)
- 🗺️ [StakeNodes' Explorer](https://explorer.pocket.network/pocket-alpha)
- 🗺️ [Soothe's Explorer](https://shannon-alpha.trustsoothe.io/)
- 👨‍💻 [Soothe's GraphQL Playground](https://shannon-alpha-api.trustsoothe.io/)

<!-- TODO_MAINNET_MIGRATION(@bryanchriswhite): Add a link to the MACT Faucet once it's live. -->

## RPC Endpoints

We provide `gRPC`, `JSON-RPC` and `REST` endpoints, which are available here:

- **RPC**: `https://shannon-testnet-grove-rpc.alpha.poktroll.com`
- **gRPC**: `https://shannon-testnet-grove-grpc.alpha.poktroll.com`
- **REST**: `https://shannon-testnet-grove-api.alpha.poktroll.com`

### Alpha JSON-RPC Example

Using `curl`:

```bash
curl -X POST https://shannon-testnet-grove-seed-rpc.alpha.poktroll.com/block
```

Using the `pocketd` binary:

```bash
pocketd query block --type=height 1 --node https://shannon-testnet-grove-seed-rpc.alpha.poktroll.com
```

## Alpha Genesis

The genesis file for the Pocket Network is located at [pokt-network-genesis](https://github.com/pokt-network/pocket-network-genesis/tree/master/shannon/testnet-alpha).
