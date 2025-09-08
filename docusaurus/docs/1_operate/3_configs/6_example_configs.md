---
title: Example Configs
sidebar_position: 6
---

This document is intended to serve as a reference for various example configurations.

## RelayMiner

### Pocket RelayMiner exposing `REST`, `CometBFT`, `JSON-RPC`, and `WebSocket`

```yaml
config:
  default_signing_key_names: [mainnet-prod-relayminer]
  smt_store_path: ":memory:" # /home/pocket/.pocket
  pocket_node:
    query_node_rpc_url: http://<YOUR_POKT_NODE>.<YOUR_TLD>:443
    query_node_grpc_url: tcp://<YOUR_POKT_NODE>.<YOUR_TLD>:443
    tx_node_rpc_url: http://<YOUR_POKT_NODE>.<YOUR_TLD>:443
  suppliers:
    - service_id: pocket
      listen_url: http://0.0.0.0:8545
      service_config:
        backend_url: https://<YOUR_POKT_NODE>.<YOUR_TLD>:443
        publicly_exposed_endpoints:
          - <PUBLICLY_EXPOSED_SUBDOMAIN>.<YOUR_TLD>.com
      rpc_type_service_configs:
        rest:
          backend_url: http://<YOUR_INTERNALIP>:1317
        comet_bft:
          backend_url: http://<YOUR_INTERNALIP>:26657
        json_rpc:
          backend_url: http://<YOUR_INTERNALIP>:8545
        websocket:
          backend_url: http://<YOUR_INTERNALIP>:48546
```
