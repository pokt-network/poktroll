---
title: MainNet
sidebar_position: 1
---

- 🚰 [StakeNode's MACT Faucet for claiming POKT](https://faucet.pocket.network)
- 🗺️ [Pocket Network Explorer](https://explorer.pocket.network/)
- 🗺️ [Poktscan Explorer](https://poktscan.pocket.network/)
- 🗺️ [Blockval's Explorer](https://explorer.blockval.io/pocket)
- 🗺️ [Chaintools's Explorer](https://explorer.chaintools.tech/pocket-network)
- 👨‍💻 [PocketDex GraphQL Playground](https://data.pocket.network/)

<!-- TODO_MAINNET_MIGRATION(@bryanchriswhite): Add a link to the MACT Faucet once it's live. -->

## RPC Endpoints

We provide `gRPC`, `JSON-RPC` and `REST` endpoints, which are available here:

- **RPC**: `https://sauron-rpc.infra.pocket.network`
- **RPC**: `https://rpc-pocket.blockval.io`
- **gRPC**: `https://sauron-grpc.infra.pocket.network`
- **REST**: `https://sauron-api.infra.pocket.network`
- **REST**: `https://api-pocket.blockval.io`

### MainNet JSON-RPC Example

Using `curl`:

```bash
curl -X POST https://sauron-rpc.infra.pocket.network/block
```

Using the `pocketd` binary:

```bash
pocketd query block --type=height 69 --network=main
```

## MainNet Genesis

The genesis file for the Pocket Network is located at [pokt-network-genesis](https://github.com/pokt-network/pocket-network-genesis/tree/master/shannon/mainnet).
