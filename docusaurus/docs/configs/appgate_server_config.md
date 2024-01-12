---
title: AppGateServer config
sidebar_position: 2
---

# AppGateServer config

## `appgateserver/config/appgate_configs_reader`

_This document describes the configuration options available through the
`appgate_server_config.yaml` file. It drives how the `AppGateServer` is setup in terms
of Pocket network connectivity, whether it acts as a self serving `Application`
or a `Gateway` to other `Applications`, and the host it listens on for incoming
`RelayRequests`._

- [AppGateServer config](#appgateserver-config)
  - [`appgateserver/config/appgate_configs_reader`](#appgateserverconfigappgate_configs_reader)
- [Usage](#usage)
- [Configuration](#configuration)
  - [`query_node_rpc_url`](#query_node_rpc_url)
  - [`query_node_grpc_url`](#query_node_grpc_url)
  - [`self_signing`](#self_signing)
  - [`signing_key`](#signing_key)
  - [`listening_endpoint`](#listening_endpoint)

# Usage

The `AppGateServer` start command accepts a `--config` flag that points to a
configuration `.yaml` file that will be used to initialize the `AppGateServer`.

```bash
pokt appgate-server start --config ./appgate_server_config.yaml --keyring-backend test
```

# Configuration

The `AppGateServer` configuration file is a `.yaml` file that contains the
following fields:

```yaml
query_node_rpc_url: tcp://<hostname>:<port>
query_node_grpc_url: tcp://<hostname>:<port>
self_signing: <boolean>
signing_key: <string>
listening_endpoint: http://<hostname>:<port>
```

## `query_node_rpc_url`
_`Required`_

The RPC URL of the Pocket node that allows the `AppGateServer` to subscribe to
events via websockets. It is re-formatted as `ws://<hostname>:<port>/websocket`
and establishes a persistent connection to the Pocket node in order to stream
events such as latest blocks and (un)delegation events.

## `query_node_grpc_url`
_`Required`_

The gRPC URL of the Pocket node that allows the `AppGateServer` to fetch data from
the Pocket network (eg. Sessions, Accounts, Applications, etc...).

## `self_signing`
_`Optional`_

Indicates whether the `AppGateServer` acts as a self serving `Application` or a
`Gateway` to other `Application`s.

If `true`, the `AppGateServer` will act as an `Application` and will only use
its address to generate a ring-signer for signing `RelayRequest`s before
forwarding them to a `RelayMiner`.

If `false`, the `AppGateServer` will act as a `Gateway` and will generate a
ring-signer from both its address and the `Application`'s address provided in
the request's `senderAddr` query parameter then use it to sign the `RelayRequests`
before forwarding them to a `RelayMiner`.

## `signing_key`
_`Required`_

Name of the key used to derive the public key and the corresponding address
for cryptographic rings generation used to sign `RelayRequests`.

The key name must be present in the keyring that is specified when the
`AppGateServer` is started.

## `listening_endpoint`
_`Required`_

The endpoint that the `AppGateServer` will listen on for incoming requests.