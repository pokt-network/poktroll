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
- [Examples](#examples)
  - [Proxy -\> Supplier referencing](#proxy---supplier-referencing)
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

The type of the proxy server to be started. Currently only `http` is supported.
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

## `type`
_`Required`_

The transport type that the service will be offered on. It must match the `type` field
of the proxy that the supplier is referencing through `proxy_names`. Currently only
`http` is supported but when other types are supported, the `type` field could
determine if additional configuration options are allowed be them optional or
required.

## `service_config`
_`Required`_

The `service_config` section of the supplier configuration is a set of options
that are specific to the service that the `RelayMiner` will be offering to the
Pocket network.

### `url`
_`Required`_

The URL of the service that the `RelayMiner` will forward the requests to when
a relay is received. It must be a valid URL (not just a host) and must be reachable
from the `RelayMiner` instance.

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

It is used to determine which RPC-Type the request is for and must be unique
across all the `hosts` lined to a given proxy.

_Note: The `name` of the supplier is automatically added to the `hosts` list_

## `proxy_names`
_`Required`_, _`Unique` within the `proxy_names` list_

The `proxy_names` section of the supplier configuration is the list of proxies
that the `RelayMiner` will use to serve the requests for the given supplier entry.

It must be a valid proxy name that is defined in the `proxies` section of the
configuration file, must be unique across the supplier's `proxy_names`  and the
`supplier` `type` must match the `type` of the referenced `proxy`.

# Examples

## Proxy -> Supplier referencing

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

## Full config example

Below is a full and commented example of a `RelayMiner` configuration file.

```yaml
# TODO_CONSIDERATION: We don't need this now, but it would be beneficial if the
# logic handling this config file could be designed in such a way that it allows for
# "hot" config changes in the future, meaning changes without restarting a process.
# This would be useful for adding a proxy or a supplier without interrupting the service.

# Name of the key (in the keyring) to sign transactions
signing_key_name: supplier1
# Relative path (on the relayminer's machine) to where the data backing
# SMT KV store exists on disk
smt_store_path: smt_stores

pocket_node:
  # Pocket node URL that exposes CometBFT JSON-RPC API.
  # This can be used by the Cosmos client SDK, event subscriptions, etc...
  query_node_rpc_url: tcp://sequencer-poktroll-sequencer:36657
  # Pocket node URL that exposes the Cosmos gRPC service, dedicated to querying purposes.
  # If unspecified, defaults to `tx_node_grpc_url`.
  query_node_grpc_url: tcp://sequencer-poktroll-sequencer:36658
  # Pocket node URL that exposes the Cosmos gRPC service.
  tx_node_grpc_url: tcp://sequencer-poktroll-sequencer:36658

# Proxies are endpoints that expose different suppliers to the internet.
proxies:
    # Name of the proxy. It will be used to reference in a supplier. Must be unique.
    # Required.
    # TODO_CONSIDERATION: if we enforce DNS compliant names, it can potentially
    # become handy in the future.
    # More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#dns-subdomain-names
  - name: http-example
    # Type of proxy: currently only http is supported but will support more
    # (https, tcp, quic ...) in the future.
    # MUST match the type of the supplier.
    # Required.
    type: http
    # Hostname to open port on. Use 0.0.0.0 in containerized environments,
    # 127.0.0.1 with a reverse-proxy when there's another process on localhost
    # that can be used as a reverse proxy (nginx, apache, traefik, etc.).
    # Required.
    host: 127.0.0.1:8080

  # TODO_IMPROVE: https is not currently supported, but this is how it could potentially look.
  # - name: example-how-we-can-support-https
  #   type: https
  #   host: 0.0.0.0:8443
  #   tls:
  #     enabled: true
  #     certificate: /path/to/crt
  #     key: /path/to/key

# Suppliers are different services that can be offered through RelayMiner.
# When a supplier is configured to use a proxy and staked appropriately,
# the relays will start flowing through RelayMiner.
suppliers:
    # Name of the supplier offered to the network.
    # Must be unique.
    # Required.
  - name: ethereum
    # Type of how the supplier offers service through the network.
    # Must match the type of the proxy the supplier is connected to.
    # Required.
    type: http
    # Configuration of the service offered through RelayMiner.
    service_config:
      # URL RelayMiner proxies the requests to.
      # Required.
      url: http://anvil.servicer:8545
      # Authentication for the service.
      # HTTP Basic Auth: https://developer.mozilla.org/en-US/docs/Web/HTTP/Authentication
      # Optional.
      authentication:
        username: user
        password: pwd

      # TODO_IMPROVE: This is not supported in code yet,
      # but some services authenticate via a header.
      # Example, if the service requires a header like `Authorization: Bearer <PASSWORD>`
      # Authorization: Bearer <PASSWORD>
      # Optional.
      headers: {}

    # A list of hosts the HTTP service is offering.
    # When linked to the proxy, that hostname is going to be used to route the
    # request to the correct supplier.
    # That hostname is what the user should stake the supplier for.
    # Must be unique within a proxy/proxies it is set up on;
    # in other words, one proxy can't offer the same hostname more than once.
    # The `name` of the supplier is automatically added to the hosts section
    # for potential troubleshooting/debugging purposes
    # Required.
    hosts:
      - ethereum.devnet1.poktroll.com
      # - ethereum # <- this part is be added automatically.

    # Names of proxies that this supplier is connected to.
    # This MUST correspond to the name in the `proxies` section
    # in order for the supplier to be accessible to the external network.
    # Required.
    proxy_names:
      - http-example # when the RelayMiner server builder runs.
  - name: 7b-llm-model
    type: http
    service_config:
      url: http://llama-endpoint
    hosts:
      - 7b-llm-model.devnet1.poktroll.com
      # - 7b-llm-model # <- this part can be added automatically.
    proxy_names:
      - http-example
```