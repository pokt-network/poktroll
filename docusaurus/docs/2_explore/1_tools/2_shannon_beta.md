---
title: Beta TestNet
sidebar_position: 2
---

## Explorers

- 🚰 [MACT Faucet for claiming tokens](https://faucet.beta.testnet.pokt.network/mact/)
- 🚰 [POKT Faucet Beta](https://faucet.beta.testnet.pokt.network/pokt/)
- 🗺️ [Poktscan beta Explorer](https://poktscan.beta.pocket.network/)
- 👨‍💻 [Beta's GraphQL Playground](https://data.beta.pocket.network/)

## RPC Endpoints

We provide `gRPC`, `JSON-RPC` and `REST` endpoints, which are available here:

- **RPC**: `https://sauron-rpc.beta.infra.pocket.network`
- **gRPC**: `https://sauron-grpc.beta.infra.pocket.network`
- **REST**: `https://sauron-api.beta.infra.pocket.network`

### Beta JSON-RPC Example

Using `curl`:

```bash
curl -X POST https://sauron-rpc.beta.infra.pocket.network/block
```

Using the `pocketd` binary:

```bash
pocketd query block --type=height 0 --network=beta
```

## Alpha Genesis

The genesis file for the Pocket Network is located at [pokt-network-genesis](https://github.com/pokt-network/pocket-network-genesis/tree/master/shannon/testnet-beta).
