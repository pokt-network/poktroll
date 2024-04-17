---
title: RelayMiner config
sidebar_position: 1
---

# `relayer/config/relayminer_configs_reader` <!-- omit in toc -->

_This document describes the configuration options available through the
`relayminer_config.yaml` file. It drives how the `RelayMiner` is setup in terms
of Pocket network connectivity, the servers it starts, which domains it accepts
queries from and which services it forwards requests to._

- [Usage](#usage)
- [Structure](#structure)
- [Top level options](#top-level-options)
  - [`signing_key_name`](#signing_key_name)
  - [`smt_store_path`](#smt_store_path)
  - [`metrics`](#metrics)
  - [`pprof`](#pprof)
- [Pocket node connectivity](#pocket-node-connectivity)
  - [`query_node_rpc_url`](#query_node_rpc_url)
  - [`query_node_grpc_url`](#query_node_grpc_url)
  - [`tx_node_rpc_url`](#tx_node_rpc_url)
- [RelayMiner proxies](#relayminer-proxies)
  - [`proxy_name`](#proxy_name)
  - [`type`](#type)
  - [`host`](#host)
- [Suppliers](#suppliers)
  - [`service_id`](#service_id)
  - [`type`](#type-1)
  - [`service_config`](#service_config)
    - [`url`](#url)
    - [`authentication`](#authentication)
    - [`headers`](#headers)
  - [`hosts`](#hosts)
  - [`proxy_names`](#proxy_names)
- [Proxy to Supplier referencing](#proxy-to-supplier-referencing)
- [RelayMiner config -\> On-chain service relationship](#relayminer-config---on-chain-service-relationship)
- [Full config example](#full-config-example)
- [Supported proxy types](#supported-proxy-types)

## Usage

The `RelayMiner` start command accepts a `--config` flag that points to a configuration
`yaml` file that will be used to setup the `RelayMiner` instance.

```bash
poktrolld relayminer --config ./relayminer_config.yaml --keyring-backend test
```

## Structure

The `RelayMiner` configuration file is a `yaml` file that contains `top level options`,
`proxies` and `suppliers` sections.

## Top level options

```yaml
signing_key_name: <string>
smt_store_path: <string>
```

### `signing_key_name`

_`Required`_

The name of the key that will be used to sign transactions, derive the public key
and the corresponding address. This key name must be present in the keyring that is used
to start the `RelayMiner` instance.

### `smt_store_path`

_`Required`_

The relative or absolute path to the directory where the `RelayMiner` will store
the `SparseMerkleTree` data on disk. This directory is used to persist the `SMT`
in a BadgerDB KV store data files.

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

Configures a [pprof](https://github.com/google/pprof/blob/main/doc/README.md) endpoint for troubleshooting and debugging performance issues.

Example configuration:

```yaml
pprof:
  enabled: true
  addr: localhost:6060
```

You can learn how to use that endpoint on [Performance Troubleshooting](../../develop/developer_guide/performance_troubleshooting.md) page.


## Pocket node connectivity

```yaml
pocket_node:
  query_node_rpc_url: tcp://<hostname>:<port>
  query_node_grpc_url: tcp://<hostname>:<port>
  tx_node_rpc_url: tcp://<hostname>:<port>
```

### `query_node_rpc_url`

_`Required`_

The RPC URL of the Pocket node that allows the `RelayMiner` to subscribe to events
via websockets. It is then re-formatted as `ws://<hostname>:<port>/websocket`
and establishes a persistent connection to the Pocket node to stream events such as
latest blocks, application staking events, etc...
If unspecified, `tx_node_rpc_url` value will be used.

### `query_node_grpc_url`

_`Optional`_

The gRPC URL of the Pocket node that allows the `RelayMiner` to query/pull data from
the Pocket network (eg. Sessions, Accounts, etc...).

### `tx_node_rpc_url`

_`Required`_

The RPC URL of the Pocket node that allows the `RelayMiner` to broadcast transactions to the a Pocket network Tendermint node.
It may have a different host than the `query_node_rpc_url` but the same value is
acceptable too.

## RelayMiner proxies

The `proxies` section of the configuration file is a list of proxies that the
`RelayMiner` will use to start servers by listening on the configured host.

At least one proxy is required for the `RelayMiner` to start.

```yaml
proxies:
  - proxy_name: <string>
    type: <enum{http}>
    host: <host>
```

### `proxy_name`

_`Required`_, _`Unique`_

Is the name of the proxy which will be used as a unique identifier to reference
proxies in the [Suppliers](#suppliers) section of the configuration file.
It corresponds to a server that will be started by the `RelayMiner` instance
and must be unique across all proxies.

### `type`

_`Required`_

The type of the proxy server to be started. Must be one of the [supported types](#supported-proxy-types).
When other types are supported, the `type` field could determine if additional
configuration options are allowed be them optional or required.

### `host`

_`Required`_, _`Unique`_

The host to which the proxy server will be started and listening to. It must be
a valid host according to the `type` filed and must be unique across all proxies.

## Suppliers

The `suppliers` section of the configuration file is a list of suppliers that
represent the services that the `RelayMiner` will offer to the Pocket network
through the configured proxies and their corresponding services to where the
requests will be forwarded to.

Each suppliers entry's `service_id` must reflect the on-chain `Service.Id` the supplier
staked for and the `hosts` list must contain the same endpoints hosts that the
supplier advertised when staking for that service.

At least one supplier is required for the `RelayMiner` to be functional.

```yaml
suppliers:
  - service_id: <string>
    type: <enum{http}>
    service_config:
      url: <url>
      authentication:
        username: <string>
        password: <string>
      headers:
        <key>: <value>
    hosts:
      - <host>
    proxy_names:
      - <string>
```

### `service_id`

_`Required`_, _`Unique`_

The Id of the service which will be used as a unique identifier to reference
a service provided by the `Supplier` and served by the `RelayMiner` instance.

It must match the `Service.Id` specified by the supplier when staking for the
service.

### `type`

_`Required`_

The transport type that the service will be offered on. It must match the `type` field
of the proxy that the supplier is referencing through `proxy_names`.
Must be one of the [supported types](#supported-proxy-types).

### `service_config`

_`Required`_

The `service_config` section of the supplier configuration is a set of options
that are specific to the service that the `RelayMiner` will be offering to the
Pocket network.

#### `url`

_`Required`_

The URL of the service that the `RelayMiner` will forward the requests to when
a relay is received, also known as **data node** or **service node**.
It must be a valid URL (not just a host) and be reachable from the `RelayMiner` instance.

#### `authentication`

_`Optional`_

The `authentication` section of the supplier configuration is a pair of `username`
and `password` that will be used by the basic authentication mechanism to authenticate
the requests that are forwarded to the service.

#### `headers`

_`Optional`_

The `headers` section of the supplier configuration is a set of key-value pairs
that will be added to the request headers when the `RelayMiner` forwards the
requests to the service. It can be used to add additional headers like
`Authorization: Bearer <TOKEN>` for example.

### `hosts`

_`Required`_, _`Unique` for each referenced proxy_, _`Unique` within the supplier's `hosts` list_

The `hosts` section of the supplier configuration is a list of hosts that the
`RelayMiner` will accept requests from. It must be a valid host that reflects
the on-chain supplier staking service endpoints.

It is used to determine if the incoming request is allowed to be processed by
the referenced proxy server as well as to check if the request's RPC-Type matches
the on-chain endpoint's RPC-Type.

There are various reasons to having multiple hosts for the same supplier services.

- The on-chain Supplier may provide the same Service on multiple domains
  (e.g. for different regions).
- The operator may want to route requests of different RPC types to
  the same proxy
- Migrating from one domain to another. Where the operator could still
  accept requests on the old domain while the new domain is being propagated.
- The operator may want to have a different domain for internal requests.
- The on-chain Service configuration accepts multiple endpoints.

It must be unique across all the `hosts` lined to a given proxy.

_Note: The `service_id` of the supplier is automatically added to the `hosts` list as
it may help troubleshooting the `RelayMiner` and/or send requests internally
from a k8s cluster for example._

### `proxy_names`

_`Required`_, _`Unique` within the `proxy_names` list_

The `proxy_names` section of the supplier configuration is the list of proxies
that the `RelayMiner` will use to serve the requests for the given supplier entry.

It must be a valid proxy name that is defined in the `proxies` section of the
configuration file, must be unique across the supplier's `proxy_names` and the
`supplier` `type` must match the `type` of the referenced `proxy`.

## Proxy to Supplier referencing

To illustrate how the `suppliers.proxy_names` and `proxies.proxy_name` fields are used
to reference proxies and suppliers, let's consider the following configuration file:

```yaml
proxies:
  - proxy_name: http-example
    ...
  - proxy_name: http-example-2
suppliers:
  - service_id: ethereum
    ...
    proxy_names:
      - http-example
      - http-example-2
  - name: 7b-llm-model
    ...
    proxy_names:
      - http-example
```

In this example, the `ethereum` supplier is referencing two proxies, `http-example`
and `http-example-2` and the `7b-llm-model` supplier is referencing only the
`http-example` proxy. This would result in the following setup:

```yaml
- http-example
  - ethereum
  - 7b-llm-model
- http-example-2
  - ethereum
```

## RelayMiner config -> On-chain service relationship

The following diagram illustrates how the _off-chain_ `RelayMiner` operator
config (yaml) must match the _on-chain_ `Supplier` actor service endpoints
for correct and deterministic behavior.

If these do not match, the behavior is non-deterministic and could result in
a variety of errors such as bad QoS, incorrect proxying, burning of the actor, etc...

_Assuming that the on-chain endpoints 1 and 2 have different hosts_

```mermaid
flowchart LR

subgraph "Supplier Actor (On-Chain)"
  subgraph "SupplierServiceConfig (protobuf)"
    subgraph svc1["Service1 (protobuf)"]
      svc1Id[Service1.Id]
      subgraph SupplierEndpoint
        EP1[Endpoint1]
        EP2[Endpoint2]
      end
    end
    subgraph svc2 ["Service2 (protobuf)"]
      svc2Id[Service2.Id]
    end
  end
end

subgraph "RelayMiner Operator (Off-Chain)"
  subgraph "DevOps Operator Configs (yaml)"
    subgraph svc1Config ["Service1 Config (yaml)"]
      svc1IdConfig[service_id=Service1.Id]-->svc1Id
      subgraph Hosts
        H1[Endpoint1.Host]-->EP1
        H2[Endpoint2.Host]-->EP2
        H3[Internal Host]
      end
    end
    subgraph svc2Config ["Service2 Config (yaml)"]
      svc2IdConfig[Service2.Id]
    end
  end
end

svc2Config-->svc2
```

## Full config example

A full and commented example of a `RelayMiner` configuration file can be found
at [localnet/poktrolld/config/relayminer_config_full_example.yaml](https://github.com/pokt-network/poktroll/tree/main/localnet/poktrolld/config/relayminer_config_full_example.yaml)

---

## Supported proxy types

The list of supported proxy types can be found at [pkg/relayer/config/types.go](https://github.com/pokt-network/poktroll/tree/main/pkg/relayer/config/types.go#L8)
