# JSON-RPC

This document outlines the complex nature of unmarshalling request payloads and
the lessons learned from attempting to do so.

## Request Structure

The JSON-RPC specification is fairly straightforward. However, the issue in the
request structure comes from a single field, the `params`. According to the
[specification](https://www.jsonrpc.org/specification#request_object) the
`params` field can be either a list or a map.

### List Params

When the parameters are a list they are to be index specifically so they align
with the expected keys they represent.

### Map Params

When the parameters are a map they are keyed by strings according to the
parameter's name.

## Parameter Type Issues

The first issue arises in how we unmarshal the JSON request into the payload's
protobuf. The following method of using a `oneof` seems like a good idea.

```protobuf
// JSONRPCRequestPayload contains the payload for a JSON-RPC request.
// See https://www.jsonrpc.org/specification#request_object for more details.
message JSONRPCRequestPayload {
    uint32 id = 1; // Identifier established by the Client to create context for the request.
    string jsonrpc = 2; // Version of JSON-RPC. Must be exactly "2.0".
    string method = 3; // Method being invoked on the server.
    oneof params {
        JSONRPCRequestPayloadParamsList list_params = 4 [(gogoproto.jsontag) = "params"];
        JSONRPCRequestPayloadParamsMap map_params = 5 [(gogoproto.jsontag) = "params"];
    }
}

// JSONRPCRequestPayloadParamsList contains the list of parameters for a JSON-RPC request.
// TODO_RESEARCH: Will this always be a string list? Or should we use a more generic type?
message JSONRPCRequestPayloadParamsList {
    repeated string params = 1;
}

// JSONRPCRequestPayloadParamsMap contains the map of parameters for a JSON-RPC request.
// TODO_RESEARCH: Will this always be a string map? Or should we use a more generic type?
message JSONRPCRequestPayloadParamsMap {
    map<string, string> params = 1;
}
```

Unfortunately the only types that cannot directly go into a `oneof` are maps and
repeated fields.

### More Issues With Types

However this assumes the maps and lists are both on `string` values. This is not
always the case. They could be any value. As such it seems appropriate to use
the `google/protobuf/any.proto` type as the value. This however, brings a new
wave of issues, as this `Any` protobuf type does not align with the `any` golang
type which means there must be some more complex logic around unmarshalling the
parameters field as each entry must be converted into the protobuf `Any` type.

## Interim Unmarshalling

These issues lead to an unfortunate consequence. We cannot directly unmarshal
the `params` field directly into the `oneof` field and we cannot directly
unmarshal the `params` field's values into `Any` types either. This is because
the `params` field is not an encoded `Any` type nor a serialised
`isJSONRPCRequestPayload_Params` type. This means we must have some sort of
interim unmarshalling where we extract everything from the request except the
`params` field and handle that seperately with more involved type checking.

The following is an example of how this would be done (ignoring the params field
is not just for string values):

```go
// Create the relay request payload.
relayRequestPayload := &types.RelayRequest_JsonRpcPayload{}
jsonPayload := &types.JSONRPCRequestPayload{}

// Unmarshal the request body bytes into an InterimJSONRPCRequestPayload.
interimPayload := &InterimJSONRPCRequestPayload{}
if err := json.Unmarshal(payloadBz, interimPayload); err != nil {
	return err
}
jsonPayload.Jsonrpc = interimPayload.Jsonrpc
jsonPayload.Method = interimPayload.Method
jsonPayload.Id = interimPayload.ID

// Set the relay json payload's JSON RPC payload params.
var mapParams map[string]string
if err := json.Unmarshal(interimPayload.Params, &mapParams); err == nil {
	jsonPayload.Params = &types.JSONRPCRequestPayload_MapParams{
		MapParams: &types.JSONRPCRequestPayloadParamsMap{
			Params: mapParams,
		},
	}
} else {
	// Try unmarshaling into a list
	var listParams []string
	if err := json.Unmarshal(interimPayload.Params, &listParams); err == nil {
		jsonPayload.Params = &types.JSONRPCRequestPayload_ListParams{
			ListParams: &types.JSONRPCRequestPayloadParamsList{
				Params: listParams,
			},
		}
	} else {
		// Neither a map nor a list
		return ErrAppGateHandleRelay.Wrapf("params must be either a map or a list of strings: %v", err)
	}
}

// Set the relay request payload's JSON RPC payload.
relayRequestPayload.JsonRpcPayload = jsonPayload
```

This is very complex and ugly.

## The Solution

The above protobuf and unmarshalling logic is not ideal and adds lots of
complexity that is undesired. Instead we must assess what we can do instead.

Ultimately we do not do anything with the request payload and as such it should
not need to be unmarshalled. This is because after we unmarshal the payload, and
include it in the `RelayRequest` structure and sign it. We immediately marshal
it and send it from the application/gateway to the supplier. They, in turn,
unmarshal the request (which involves unmarshalling the payload) and verify the
signature just to marshal the payload once more and send it to the service.

This leads to the natural conclusion that instead of this complex (un)marshalling
logic we should instead just store the payload as its natively encoded `[]byte`
state and pass this into the `RelayRequest` and avoid unmarshalling it at all.
Doing this brings numerous benefits:

1. Less complexity
2. Increased privacy for the user (at surface level at least)
    - We do not touch their request or even attempt to decode it

This would mean the `RelayRequest` would look like the following:

```protobuf
// RelayRequest holds the request details for a relay.
message RelayRequest {
    RelayRequestMetadata meta = 1;
    bytes payload = 2;
}
```

### Future Relay Pricing

Doing the above may bring questions as to how we could implement different relay
pricing mechanisms, such as: compute units, cost per byte, etc.

In fact this simplifies this logic. For compute units we could take the following
approach:

1. Partially unmarshal the payload minus the `params` field
2. Assign a weight to the request based on the `method` field
    - The `method` field would be weighted in such a manner that accounts for
    the types and length of parameters it requires and means we do not need
    to touch the `params` field at all

For cost per byte we could simply take the length of the payload which is its
number of bytes and act accordingly.

### Different Request Types

This document has focussed on JSON-RPC which is at the time of writing the only
request type the Pocket Network Shannon upgrade supports. However, we aim to
support numerous types of relays: gRPC, REST, GraphQL, etc. These would each
have their own unmarshalling intricacies which would require their own complex
logic to properly define a protobuf that could encapsulate every payload type
and how we could unmarshal into this type.

This is not ideal. Instead following the same logic as above we should store
any and all payloads in their serialised forms within the `RelayRequest` as
this is what is needed to send the request through to the servicer. Again
we would be able to extract just the necessary `method` field equivalents for
each payload type and define compute units accordingly and cost per bytes is
the most simple.
