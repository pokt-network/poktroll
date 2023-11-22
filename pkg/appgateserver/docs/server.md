# AppGateServer

<!-- toc -->

- [Overview](#overview)
  - [Incoming Requests](#incoming-requests)
  - [Chain Interactions](#chain-interactions)
  - [Responses](#responses)
- [Running the `AppGateServer`](#running-the-appgateserver)
  - [Applications](#applications)
  - [Gateways](#gateways)
- [Requests](#requests)
  - [Request Types](#request-types)
    - [Synchronous Protocols](#synchronous-protocols)
    - [Asynchronous Protocols](#asynchronous-protocols)
- [Sessions](#sessions)
- [Endpoint Selection](#endpoint-selection)
- [Signatures](#signatures)
  - [Rings](#rings)
    - [Ring Signatures](#ring-signatures)
  - [Response Verification](#response-verification)

<!-- tocstop -->

## Overview

The `AppGateServer` is a proxy server that listens for incoming request and
handles the interactions with the chain and suppliers in order to service the
request. This server can act as either an application or gateway actor,
depending on how it is configured when started.

See [appgateserver](../) for the specifics of the `AppGateServer`'s
implementation.

### Incoming Requests

The server will listen on a configured endpoint for incoming requests from an
application (or applications if it is in "gateway mode"). The requests can be
of any type, provided the protocol is "synchronous", that is there is a
one-to-one correspondance between requests and responses.

See [requests](#requests) for more detail on requests, thier structure and
how they are formatted.

### Chain Interactions

The server will handle all chain interactions, so the application need only
send their requests and receive their responses. These include:

1. Session Retrieval (see [sessions](#sessions))
1. Supplier Endpoint Selection (see [endpoint selection](#endpoint-selection))
1. Ring Creation and Caching (see [rings](#rings))
1. Response Verification (see [response verification](#response-verification))

### Responses

The server will pass along the payload received from the request through to the
supplier that will execute the request and return the response. The server will
then pass this payload back in the same format as the request (JSON-RPC, REST,
gRPC, etc.). If an internal error occurs at any stage of carrying out the request
the `AppGateServer` will construct an error payload that is of the same format
as the request and return this, along with an error message detailing what
issue occured.

## Running the `AppGateServer`

In order to run an `AppGateServer`, the user must first stake either their
application or gateway accounts.

_NOTE: The `AppGateServer` will still start without the actor being staked but
will not be usable_

### Applications

To start the `AppGateServer` as an application the following command must be ran:

```sh
poktrolld --home={CHAIN_HOME_DIR} appgate-server --signing-key={SIGNING KEY NAME} \
    --self-signing --listening-endpoint={ENDPOINT TO LISTEN ON} \
    --query-node={ENDPOINT OF A POCKET NODE TO QUERY ON CHAIN DATA} \
    --keyring-backend={KEYRING}
```

Here the `signing-key` is the name of the key in the `keyring-backend` keyring.
This is used to sign relays. If the `self-signing` flag is set this means the
`AppGateServer` will store the address associated with the `signing-key`'s key
and use this to retrieve its own ring and sign relays using its own key and ring.

The `query-node` flag is the flag of a trusted Pocket node that will be used to
query the on-chain state for session retrieval, ring creation and public key
retrieval for response verification.

The `listening-endpoint` flag determines the endpoint the `AppGateServer` will
listen on to receive requests from the application. This is the endpoint the
application will send their requests to.

### Gateways

Starting an `AppGateServer` as a gateway works similarly to starting an
[application](#applications). The only difference is the omission of the
`self-signing` flag.

```sh
poktrolld --home={CHAIN_HOME_DIR} appgate-server --signing-key={SIGNING KEY NAME} \
    --listening-endpoint={ENDPOINT TO LISTEN ON} \
    --query-node={ENDPOINT OF A POCKET NODE TO QUERY ON CHAIN DATA} \
    --keyring-backend={KEYRING}
```

When started without the `self-signing` flag the `AppGateServer` will not store
the address associated with the `signing-key` flag. This means the requests that
it receives **must** include the application's address in the
[endpoint](#requests).

When a gateway receives a request it will fetch the ring of the application
sending the request. It will then sign the request using `signing-key` used to
start the gateway.

_NOTE: The gateway is only able to sign relays for applications who have
delegated to the gateway_

## Requests

Requests from an application are received on the `AppGateServer`'s listening
address. This acts as a single endpoint the application can send their requests
to.

When sending a request the service **must** be specified in the endpoint and
if the `AppGateServer` is a gateway then the application's address must also be
included.

The endpoint will look like the following:

```
scheme://host:port/{service}[?senderAddr={application address}]
```

### Request Types

Depending on the service the request will be of a different form: JSON-RPC,
REST, gRPC, GraphQL, etc. These different payload types can be catagorised into
two catagories.

1. Synchronous Protocols
   - Where there is one request for each response (an `n-n` correspondance)
1. Asynchronous Protocols
   - Where there is many requests for a single response (an `m-n` correspondance)

The `AppGateServer` handles these types of protocol seperately.

#### Synchronous Protocols

Synchronous protocols are handled by simply passing the serialised payload of the
request from the application to the supplier, who in turn passes it directly to
the service itself. The response is handled in the same way.

In the "happy path" the payload is passed without error to the service and the
appropirate response is returned to the application in the format it is given
in from the service.

_NOTE: The happy path described here encompasses both valid requests and invalid
ones too. These are seen as the same at the protocol level_

In the case there is an internal server error, with the `AppGateServer` itself,
the payload must be partially unmarshalled to retrieve specific information.
This information is required for the error to be returned not only in the
correct format, but also to contain specified fields.

For example a JSON-RPC request will contain the following fields:

```json
{
    "id": 1,                        // id of the request
    "jsonrpc": "2.0"                // JSON-RPC version number
    "method": "eth_getBlockNumber", // the method to be executed
    "params": {},                   // either a list or map of params
}
```

When returning an internal server error the response must contain the fields:

```json
{
    "id": 1                // this must be the same as from the request
    "jsonrpc": "2.0"       // this must be the same as from the request
    "error": {
        "code":    -32000, // the error code of an internal error
        "message": "",     // the error message
        "data":    null,   // a nil data field
    },
}
```

In order to extract the relevent fields from the serialised payload we must
partially unmarshal it, extracting only the desired fields. The same logic
is used for other types of synchronous protocols, extracting different fields for
each payload type.

#### Asynchronous Protocols

_NOTE: Asynchronous Protocols are currently unsupported as of writing this
document_

## Sessions

The `AppGateServer` will handle the interaction with the chain to retrieve the
current session for the application whose request it is handling. This enables
the application to simply send their requests without having to manage their
sessions or keep track of them.

The `AppGateServer` will query the chain for the current session ID for the
given application and service pair.

See [session.go](../session.go) for more details on the implementation of
session retrieval.

## Endpoint Selection

Endpoints for the suppliers in a session are currently chosen based on the
first suitable supplier for the service from within the session.

_NOTE: The endpoint selection algorithm is planned to factor in other factors
such as Quality of Service and move away from the greedy selection
implementation_

See [endpoint-selector.go](../endpoint_selector.go) for more details on the
implementation of supplier endpoint selection.

## Signatures

The `AppGateServer` signs all requests (in both application and gateway mode)
using the ring of the application sending the request. They sign the ring using
the private key associated with the key used when starting the server.

When a response is received from the supplier, it is verified using the public
key of the supplier who executed the request.

### Rings

Rings are created using the application's on-chain state. This includes a list
of addresses of gateways the specified application is delegated to. If an
`AppGateServer` is started in application mode, and the application has
delegated to gateways on-chain, even thought the application is self-signing,
it's ring will still contain these gateways' addresses corresponding public
keys.

In order to create the ring these addresses are first converted to their
corresponding public keys. This is achieved by querying the `auth` module.

These public keys are then converted to their points on the `secp256k1` curve
and this list of points is used to create the ring used for signing.

Rings are cached after their first creation to reduce latency on the request.
These caches are invalidated whenever the application changes their delegated
gateways. This is to ensure the `AppGateServer` has the most up to date ring
to produce valid signatures.

_NOTE: Ring cache invalidation is not currently implemented_

<!-- TODO(@h5law): Update this link when the ring cache package is merged -->

See [rings.go](../rings.go) for more details on the implementation of ring
creation.

#### Ring Signatures

In order to sign the the relay request the `AppGateServer`'s public key must
be included in the ring for an application. Each application's ring contains
at least its own public key to allow for self signing of requests. For a gateway
to sign on behalf of an application it must have been delegated to on-chain
prior to the session starting.

_NOTE: Grace periods for delegation changes are not currently implemented_

### Response Verification

Supplier's sign the response prior to sending it back to the `AppGateServer`.
This is a simple signature using the public key of the supplier.

In order to retrieve the supplier's public key the `auth` module is queried
given the supplier's address and then their public key is extracted from the
response and used to verify the signature of the relay's response.
