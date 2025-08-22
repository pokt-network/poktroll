---
title: WebSocket Support
sidebar_position: 5
---

This page is meant to serve as a quick reference to supplement the [full RelayMiner config reference](./4_relayminer_config.md)
and the [full Supplier config](./3_supplier_config.md) to add WebSocket support to your RelayMiner and Supplier.

:::tip Supplier & RelayMiner cheatsheet

See the [Supplier & RelayMiner cheatsheet](../1_cheat_sheets/4_supplier_cheatsheet.md) for more details on running a Supplier & RelayMiner.

:::

## Base Example

The example below shows how to configure and update an existing Suppllier & RelayMiner for `base` service,
to add `WebSocket` support in addition to `JSON-RPC`.

### Onchain Supplier Configuration Example

Update your `supplier_config.yaml` to add `WebSocket` support like so:

```yaml
owner_address: <OWNER_ADDRESS>
operator_address: <OPERATOR_ADDRESS>
stake_amount: <STAKE_AMOUNT>upokt
default_rev_share_percent:
  <REWARD_ADDRESS>: 100
services:
  - service_id: "base"
    endpoints:
      - publicly_exposed_url: http://<YOUR_PUBLIC_RELAY_MINER_IP>:8545
        rpc_type: JSON_RPC
      - publicly_exposed_url: ws://<YOUR_PUBLIC_RELAY_MINER_IP>:8546
        rpc_type: WEBSOCKET
```

Then, run `pocketd tx supplier stake-supplier ...` to update your Supplier record onchain.

:::note Key Configuration

Note that the `publicly_exposed_url` of `services[].endpoints[].rpc_type: WEBSOCKET` and `services[].endpoints[].rpc_type: JSON_RPC` is exactly the same except for using the `ws` or `wss` protocol instead of `http` or `https`.

:::

### Offchain RelayMiner Configuration Example

Update your `relayminer_config.yaml` to add `WebSocket` support like so:

```yaml
default_signing_key_names:
  - supplier
smt_store_path: ":memory:" # /path/to/.pocket/smt
pocket_node:
  query_node_rpc_url: https://<RPC_NODE>
  query_node_grpc_url: https://<GRPC_NODE>:443
  tx_node_rpc_url: https://<TX_NODE>
suppliers:
  - service_id: "base"
    # Default (JSON-RPC)
    service_config:
      backend_url: "http://<YOUR_BASE_SERVICE_BACKEND_IP>:8545"
    rpc_type_service_configs:
      # JSON-RPC WebSocket
      websocket:
        backend_url: "ws://<YOUR_BASE_SERVICE_BACKEND_IP>:8546"
    listen_url: http://0.0.0.0:8545
  metrics:
    enabled: false
    addr: :9090
  pprof:
    enabled: false
```

Then, restart your RelayMiner via `pocketd relayminer start ...` to apply the changes.

:::note Key Configuration

Note that `suppliers[].rpc_type_service_configs` is needs to explicitly add and specify the `websocket.backend_url`.
The default EVM WebSocket port is `8546`, but this may vary based on your configuration.

:::
