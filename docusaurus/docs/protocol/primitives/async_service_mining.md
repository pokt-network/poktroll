# Mining Asynchronous Services

:::warning

Note: This documentation describes the behavior as of PR #1073. Implementation
details may change as the protocol evolves.

:::

The bridge represents a WebSocket bridge between the gateway and the service
backend. It handles the forwarding of relay requests from the gateway to the
service backend and relay responses from the service backend to the gateway.

## Asynchronous Message Handling

Due to the asynchronous nature of WebSockets, there isn't always a 1:1 mapping
between requests and responses. The bridge must handle two common scenarios:

### 1. Many Responses for Few Requests (M-resp >> N-req)

In this scenario, a single request can trigger multiple responses over time.
For example:
- A client subscribes once to an event stream (eth_subscribe)
- The client receives many event notifications over time through that single
  subscription

### 2. Many Requests for Few Responses (N-req >> M-resp)

In this scenario, multiple requests may be associated with fewer responses.
For example:
- A client uploads a large file in chunks, sending many requests
- The server only occasionally sends progress updates

## Design Implications

This asynchronous design has two important implications:

1. **Reward Eligibility**: Each message (inbound or outbound) is treated as a
   reward-eligible relay. For example, with eth_subscribe, both the initial
   subscription request and each received event would be eligible for rewards.

2. **Message Pairing**: To maintain protocol compatibility, the bridge must always
   pair messages when submitting to the miner. It does this by combining the most
   recent request with the most recent response.

## Future Considerations

Currently, the RelayMiner is paid for each incoming and outgoing message
transmitted. While this is the most common and trivial use case, future services
might have different payable units of work (e.g. packet size, specific packet or
data delimiter...).

To support these use cases, the bridge should be extensible to allow for custom
units of work to be metered and paid.