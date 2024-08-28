---
title: AppGateServer config
sidebar_position: 5
---

# AppGateServer config <!-- omit in toc -->

This document describes the configuration options for the `AppGateServer`,
an `Application` or `Gateway` co-processor/sidecar
that acts as the real server for querying request, signing requests and verifying responses.

This document describes the configuration options available for the
`AppGateServer`through the `appgate_server_config.yaml` file.

:::tip

You can find a fully featured example configuration at [appgate_server_config_example.yaml](https://github.com/pokt-network/poktroll/blob/main/localnet/poktrolld/config/appgate_server_config_example.yaml).

:::

- [Introduction](#introduction)
- [Usage](#usage)
- [Configuration](#configuration)
  - [`query_node_rpc_url`](#query_node_rpc_url)
  - [`query_node_grpc_url`](#query_node_grpc_url)
  - [`self_signing`](#self_signing)
  - [`signing_key`](#signing_key)
  - [`listening_endpoint`](#listening_endpoint)
  - [`metrics`](#metrics)
  - [`pprof`](#pprof)

## Introduction

It is responsible for multiple things:

1. Determines how the `AppGateServer` with respect to Pocket network connectivity
2. Whether it acts as a self serving `Application` or a `Gateway` to other `Applications`
3. Configures the host(s) it listens on for incoming `RelayRequests`

## Usage

The `AppGateServer` start command accepts a `--config` flag that points to a
configuration `.yaml` file that will be used to initialize the `AppGateServer`.

:::warning

TestNet is not ready as of writing this documentation, so you may
need to adjust the command below appropriately.

:::

```bash
poktrolld appgate-server  \
  --config ./appgate_server_config.yaml \
  --keyring-backend test
```

## Configuration

The `AppGateServer` configuration file is a `.yaml` file that contains the
following fields:

```yaml
query_node_rpc_url: tcp://<hostname>:<port>
query_node_grpc_url: tcp://<hostname>:<port>
self_signing: <boolean>
signing_key: <string>
listening_endpoint: http://<hostname>:<port>
metrics:
  enabled: true
  addr: :9090
```

### `query_node_rpc_url`

_`Required`_

The RPC URL of the Pocket node that allows the `AppGateServer` to subscribe to
on-chain CometBFT events via websockets. It is re-formatted by the SDK as
`ws://<hostname>:<port>/websocket` and establishes a persistent connection to
the Pocket Node in order to stream events such as latest blocks, and other
information such as on-chain (un)delegation events.

### `query_node_grpc_url`

_`Required`_

The gRPC URL of the Pocket node that allows the `AppGateServer` to fetch data
from the Pocket network (eg. Sessions, Accounts, Applications, etc...).

### `self_signing`

:::tip

tl;dr

- `true` -> `AppGateServer` acts as an `Application`
- `false` -> `AppGateServer` acts as a `Gateway`

:::

_`Optional`_

Indicates whether the `AppGateServer` acts as a self serving `Application` or a
`Gateway` to other `Application`s.

If `true`, the `AppGateServer` will act as an `Application` and will only use
its own address to generate a ring-signer for signing `RelayRequest`s before
forwarding them to a `RelayMiner`.

If `false`, the `AppGateServer` will act as a `Gateway` and will generate a
ring-signer from both its address and the `Application`'s address provided in
the request's `applicationAddr` query parameter then use it to sign the `RelayRequests`
before forwarding them to a `RelayMiner`.

### `signing_key`

_`Required`_

Name of the key used to derive the public key and the corresponding address
for cryptographic rings generation used to sign `RelayRequests`.

The key name must be present in the keyring that is specified when the
`AppGateServer` is started.

### `listening_endpoint`

_`Required`_

The endpoint that the `AppGateServer` will listen on for incoming requests.

### `metrics`

_`Optional`_

This section configures a Prometheus exporter endpoint, enabling the collection
and export of metrics data. The `addr` field specifies the network address for
the exporter to bind to. It can be either a port number, which assumes binding
to all interfaces, or a specific host:port combination.

Example configuration:

```yaml
metrics:
  enabled: true
  addr: :9090
```

When `enabled` is set to `true`, the exporter is active. The addr `value` of
`:9090` implies the exporter is bound to port 9090 on all available network
interfaces.

### `pprof`

_`Optional`_

Configures a [pprof](https://github.com/google/pprof/blob/main/doc/README.md)
endpoint for troubleshooting and debugging performance issues.

Example configuration:

```yaml
pprof:
  enabled: true
  addr: localhost:6060
```

You can learn how to use that endpoint on the [Performance Troubleshooting](../../develop/developer_guide/performance_troubleshooting.md) page.
