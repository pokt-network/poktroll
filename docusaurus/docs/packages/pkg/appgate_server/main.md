---
title: AppGate Server
sidebar_position: 1
---

# AppGateServer <!-- omit in toc -->

- [What is AppGate Server?](#what-is-appgate-server)
  - [Starting the AppGateServer](#starting-the-appgateserver)
- [AppGateServer as an Application](#appgateserver-as-an-application)
  - [RPC request schema](#rpc-request-schema)
- [Gateway](#gateway)
  - [RPC request schema](#rpc-request-schema-1)
- [POKTRollSDK integration](#poktrollsdk-integration)

## What is AppGate Server?

`AppGate Server` is a ready to use component that allows `Application`s and
`Gateway`s to relay RPC requests to the Pocket Network `Supplier`s without having
to manage the underlying logic of the protocol.

An operator only needs to specify a single [configuration file](configs/appgate_server_config.md),
in order to run a sovereigen `Application` or a `Gateway` via an `AppGate Server`.

### Starting the AppGateServer

`AppGateServer` could be started by running the following command:

```bash
poktrolld appgate-server  \
  --config <config-file> \
  --keyring-backend <keyring-type>
```

Where `<config-file>` is the path to the `.yaml` [configuration file](configs/appgate_server_config.md)
and `<keyring-type>` is the type of keyring to use.

Launching the `AppGateServer` starts an HTTP server that listens for incoming
RPC requests and forwards them to the appropriate Pocket Network `Supplier`s.

It takes care of:

- Querying and updating the list of `Supplier`s that are allowed to serve the
  `Application` given a `serviceId`.
- Selecting a `Supplier` to send the RPC request to.
- Appending the `Application`/`Gateway` ring-signature to the `RelayRequest`
  before sending it to the `Supplier`.
- Sending the `RelayRequest` to the `Supplier`.
- Verifying the `Supplier`'s signature.
- Returning the `RelayResponse` to the requesting client

The `AppGateServer` could be configured to act as a `Gateway` or as a `Application`

## AppGateServer as an Application

If the `self_signing` field in the
[configuration file (self_signing)](configs/appgate_server_config.md#self_signing)
is set to `true`, the `AppGateServer` will act as an `Application`; serving only
the address derived from the `signing_key` field in the
[configuration file (signing_key)](configs/appgate_server_config.md#signing_key).

`RelayRequests` sent to the `AppGateServer` will be signed with the `signing_key`
resulting in a ring-signature that contains only the `Application`'s address.

:::warning

The `AppGateServer` is able to serve RPC requests provided the `Application`
is appropriately staked for the service requested on the Pocket Network.

:::

### RPC request schema

When acting as an `Application`, the `AppGateServer` expects the RPC request to
contain the `serviceId` as an URL path parameter and the request content as
the body of the request.

The RPC request should be sent to the `listening_endpoint` specified in the
[configuration file (listening_endpoint)](configs/appgate_server_config.md#listening_endpoint).

The following `curl` command demonstrates how to send a JSON-RPC type request
to the `AppGateServer`:

```bash
curl -X POST \
  http://<hostname>:<port>/<serviceId> \
  -H 'Content-Type: application/json' \
  -d '{
    "jsonrpc": "2.0",
    "method": "<method_name>",
    "params": [],
    "id": 1
  }'
```

## Gateway

If the `self_signing` field in the
[configuration file (self_signing)](configs/appgate_server_config.md#self_signing)
is set to `false`, then the `AppGateServer` will act as a `Gateway`, serving
`Application`s that delegated to the `Gateway` address represented by the
`signing_key` field in the
[configuration file (signing_key)](configs/appgate_server_config.md#signing_key).

The `AppGateServer` will determine the `Application` address to use by extracting it
from the `senderAddr` query parameter and use it along with the `signing_key` to
generate a ring-signature that contains both the `Application`'s address and the
`Gateway`'s address.

:::warning

The `Gateway` must be appropriately staked and the `Application` must be staked
for the requested service in addition to delegating to the `Gateway` it is
interacting with.

:::

### RPC request schema

When acting as a `Gateway`, the `AppGateServer` expects the RPC request to
contain the `serviceId` as an URL path parameter, the `Application` address as
a query parameter and the request content as the body of the request.

The RPC request should be sent to the `listening_endpoint` specified in the
[configuration file (listening_endpoint)](configs/appgate_server_config.md#listening_endpoint).

The following `curl` command demonstrates how to send a JSON-RPC type request
to the `AppGateServer`:

```bash
curl -X POST \
  http://<hostname>:<port>/<serviceId>?senderAddr=<application_address> \
  -H 'Content-Type: application/json' \
  -d '{
    "jsonrpc": "2.0",
    "method": "<method_name>",
    "params": [],
    "id": 1
  }'
```

## POKTRollSDK integration

The `AppGateServer` implementation uses the [POKTRollSDK](packages/pkg/sdk/sdk.md) to
interact with the Pocket Network and could be taken as a reference for how to
integrate the `POKTRollSDK` with a custom `Application` or `Gateway` logic to send
RPC requests to the Pocket Network.

The `AppGateServer`'s own logic is responsible for:

- Exposing the HTTP server that listens for incoming RPC requests.
- Extracting the `serviceId` and `Application` address from the RPC request.
- Calling `POKTRollSDK.GetSessionSupplierEndpoints` to get the list of `Supplier`s
  that are allowed to serve the `Application`.
- Selecting a `Supplier` to send the RPC request to.
- Calling the `POKTRollSDK.SendRelay` to send the `RelayRequest` to the selected
  `Supplier`.
- Returning the verified `RelayResponse` to the RPC request sender.

While leaving the underlying Pocket Network protocol logic to the `POKTRollSDK`:

- Being up-to-date with the latest `Session`.
- Maintaining the list of `Supplier`s that are allowed to serve the `Application`.
- Forming the `RelayRequest` object.
- Creating the ring-signature for the `RelayRequest`.
- Sending the `RelayRequest` to the `Supplier`.
- Verifying the `Supplier`'s signature.

A sequence diagram demonstrating the interaction between the `AppGateServer` and
the `POKTRollSDK` can be found in the [POKTRollSDK documentation](packages/pkg/sdk/sdk.md#poktrollsdk-sequence-diagram).
