---
title: RelayMiner config
sidebar_position: 1
---

## `relayer/config/relayminer_configs_reader`

_This document describes the configuration options available through the
`relayminer_config.yaml` file. It drives how the `RelayMiner` is setup in terms
of Pocket network connectivity, the servers it starts, which domains it accepts
queries from and which services it forwards requests to._

- [Usage](#usage)
- [Structure](#structure)
- [Top level options](#top-level-options)
  - [`signing_key_name`](#signing_key_name)
  - [`smt_store_path`](#smt_store_path)
- [Pocket node connectivity](#pocket-node-connectivity)
  - [`query_node_rpc_url`](#query_node_rpc_url)
  - [`query_node_grpc_url`](#query_node_grpc_url)
  - [`tx_node_grpc_url`](#tx_node_grpc_url)
- [RelayMiner proxies](#relayminer-proxies)
  - [`name`](#name)
  - [`type`](#type)
  - [`host`](#host)
- [Suppliers](#suppliers)
  - [`name`](#name-1)
  - [`type`](#type-1)
  - [`service_config`](#service_config)
    - [`url`](#url)
    - [`authentication`](#authentication)
    - [`headers`](#headers)
  - [`hosts`](#hosts)
  - [`proxy_names`](#proxy_names)
- [Proxy -\> Supplier referencing](#proxy---supplier-referencing)
  - [RelayMiner setup -\> On-chain service relationship](#relayminer-setup---on-chain-service-relationship)
  - [Full config example](#full-config-example)

# Usage

The `RelayMiner` start command accepts a `--config` flag that points to a configuration
`yaml` file that will be used to setup the `RelayMiner` instance.

```bash
poktrolld relayminer --config ./relayminer_config.yaml --keyring-backend test
```

# Structure

The `RelayMiner` configuration file is a `yaml` file that contains `top level options`,
`proxies` and `suppliers` sections.

# Top level options

```yaml
signing_key_name: <string>
smt_store_path: <string>
```

## `signing_key_name`
 _`Required`_

The name of the key that will be used to sign transactions, derive the public key
and the corresponding address. This key name must be present in the keyring that is used
to start the `RelayMiner` instance.

## `smt_store_path`
_`Required`_

The relative or absolute path to the directory where the `RelayMiner` will store
the `SparseMerkleTree` data on disk. This directory is used to persist the `SMT`
in a BadgerDB KV store data files.

# Pocket node connectivity

```yaml
pocket_node:
  query_node_rpc_url: tcp://<host>
  query_node_grpc_url: tcp://<host>
  tx_node_grpc_url: tcp://<host>
```

## `query_node_rpc_url`
_`Required`_

The RPC URL of the Pocket node that allows the `RelayMiner` to subscribe to events
via websockets. It is then re-formatted as `ws://<host>/websocket` and establishes a
persistent connection to the Pocket node to stream events such as latest blocks,
application staking events, etc...

## `query_node_grpc_url`
_`Optional`_

The gRPC URL of the Pocket node that allows the `RelayMiner` to query/pull data from
the Pocket network (eg. Sessions, Accounts, etc...). If unspecified, `tx_node_grpc_url`
value will be used.

## `tx_node_grpc_url`
_`Required`_

The gRPC URL of the Pocket node that allows the `RelayMiner` to submit transactions.
It may have a different host than the `query_node_rpc_url` or the `tx_node_grpc_url`
but same values are acceptable too.

# RelayMiner proxies

The `proxies` section of the configuration file is a list of proxies that the
`RelayMiner` will use to start servers by listening on the configured host.

At least one proxy is required for the `RelayMiner` to start.

```yaml
proxies:
  - name: <string>
    type: <enum{http}>
    host: <host>
```

## `name`
_`Required`_, _`Unique`_

Is the name of the proxy which will be used as a unique identifier to reference
proxies in the [Suppliers](#suppliers) section of the configuration file.
It corresponds to a server that will be started by the `RelayMiner` instance
and must be unique across all proxies.

## `type`
_`Required`_

The type of the proxy server to be started. Must be one of the [supported types](https://github.com/pokt-network/poktroll/tree/main/pkg/relayer/config/types.go#L8).
When other types are supported, the `type` field could determine if additional
configuration options are allowed be them optional or required.

## `host`
_`Required`_, _`Unique`_

The host to which the proxy server will be started and listening to. It must be
a valid host according to the `type` filed and must be unique across all proxies.

# Suppliers

The `suppliers` section of the configuration file is a list of suppliers that
represent the services that the `RelayMiner` will offer to the Pocket network
through the configured proxies and their corresponding services to where the
requests will be forwarded to.

Each suppliers entry's `name` must reflect the on-chain `Service.Id` the supplier
staked for and the `hosts` list must contain the same endpoints hosts that the
supplier advertised when staking for that service.

At least one supplier is required for the `RelayMiner` to be functional.

```yaml
suppliers:
  - name: <string>
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

## `name`
_`Required`_, _`Unique`_

The name of the service which will be used as a unique identifier to reference
a service provided by the `Supplier` and served by the `RelayMiner` instance.

It must match the `Service.Id` specified by the supplier when staking for the
service.

## `type`
_`Required`_

The transport type that the service will be offered on. It must match the `type` field
of the proxy that the supplier is referencing through `proxy_names`.
Must be one of the [supported types](https://github.com/pokt-network/poktroll/tree/main/pkg/relayer/config/types.go#L8).

## `service_config`
_`Required`_

The `service_config` section of the supplier configuration is a set of options
that are specific to the service that the `RelayMiner` will be offering to the
Pocket network.

### `url`
_`Required`_

The URL of the service that the `RelayMiner` will forward the requests to when
a relay is received, also known as **data node** or **service node**.
It must be a valid URL (not just a host) and be reachable from the `RelayMiner` instance.

### `authentication`
_`Optional`_

The `authentication` section of the supplier configuration is a pair of `username`
and `password` that will be used by the basic authentication mechanism to authenticate
the requests that are forwarded to the service.

### `headers`
_`Optional`_

The `headers` section of the supplier configuration is a set of key-value pairs
that will be added to the request headers when the `RelayMiner` forwards the
requests to the service. It can be used to add additional headers like
`Authorization: Bearer <TOKEN>` for example.

## `hosts`
_`Required`_, _`Unique` for each referenced proxy_, _`Unique` within the supplier's `hosts` list_

The `hosts` section of the supplier configuration is a list of hosts that the
`RelayMiner` will accept requests from. It must be a valid host that reflects
the on-chain supplier staking service endpoints.

It is used to determine if the incoming request is allowed to be processed by
the referenced proxy server as well as to check if the request's RPC-Type matches
the on-chain endpoint's RPC-Type.

The reasons to have multiple hosts for the same supplier service are:
- The on-chain Supplier may provide the same Service on multiple domains
(e.g. for different regions).
- The operator may want to route requests of different RPC types to
the same proxy
- Migrating from one domain to another. Where the operator could still
accept requests on the old domain while the new domain is being propagated.

It must be unique across all the `hosts` lined to a given proxy.

_Note: The `name` of the supplier is automatically added to the `hosts` list as
it may help troubleshooting the `RelayMiner` and/or send requests internally
from a k8s cluster for example._

## `proxy_names`
_`Required`_, _`Unique` within the `proxy_names` list_

The `proxy_names` section of the supplier configuration is the list of proxies
that the `RelayMiner` will use to serve the requests for the given supplier entry.

It must be a valid proxy name that is defined in the `proxies` section of the
configuration file, must be unique across the supplier's `proxy_names`  and the
`supplier` `type` must match the `type` of the referenced `proxy`.

# Proxy -> Supplier referencing

To illustrate how the `proxy_names` and `name` fields are used to reference
proxies and suppliers, let's consider the following configuration file:

```yaml
proxies:
  - name: http-example
    ...
  - name: http-example-2
suppliers:
  - name: ethereum
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

```
- http-example
  - ethereum
  - 7b-llm-model
- http-example-2
  - ethereum
```
## RelayMiner setup -> On-chain service relationship

The following diagram illustrates how the `RelayMiner` config must match the
on-chain staked service endpoints.

```mermaid
flowchart RL
  subgraph PocketNetwork
    subgraph Supplier[On-chain Supplier]
      subgraph Svc1[Service-1]
        SId1[Service-1.Id]
        subgraph Endpoints
          EP1[Endpoint-1]
          EP2[Endpoint-2]
        end
      end
      subgraph Svc2[Service-2]
        SId2[Service-2.Id]
      end
      subgraph Svcs[Services...]
      end
    end
  end

  subgraph RelayMiner
    subgraph HTTP[Proxy=ProxyTypeHTTP]
      subgraph CSvc1[Service-1 Config]
        CSId1[name=Service-1.Id]-->SId1
        subgraph Hosts
          H1[Endpoint-1.Host]-->EP1
          H2[Endpoint-2.Host]-->EP2
        end
      end
      subgraph CSvc2[Service-2]
        CSId2[Service-2.Id]
      end
    end
    subgraph Proxies[Other proxy servers...]
    end
  end
  CSvc2-->Svc2
```

## Full config example

A full and commented example of a `RelayMiner` configuration file can be found
at [localnet/poktrolld/config/relayminer_config_full_example.yaml](https://github.com/pokt-network/poktroll/tree/main/localnet/poktrolld/config/relayminer_config_full_example.yaml)
