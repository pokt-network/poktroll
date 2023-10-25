# Package `pocket/pkg/client/events_query`

> An event query package for interfacing with cometbft and the Cosmos SDK, facilitating subscriptions to chain event messages.

## Overview

The `events_query` package provides a client interface to subscribe to chain event messages. It abstracts the underlying connection mechanisms and offers a clear and easy-to-use way to get events from the chain. Highlights:

- Offers subscription to chain event messages matching a given query.
- Uses the Gorilla WebSockets package for underlying connection operations.
- Provides a modular structure with interfaces allowing for mock implementations and testing.
- Offers considerations for potential improvements and replacements, such as integration with the cometbft RPC client.

## Architecture Diagrams

[Add diagrams here if needed. For the purpose of this mockup, we'll assume none are provided.]

## Installation

```bash
go get github.com/pokt-network/poktroll/pkg/client/events_query
```

## Features

- **Websocket Connection**: Uses the Gorilla WebSockets for implementing the connection interface.
- **Events Subscription**: Subscribe to chain event messages using a simple query mechanism.
- **Dialer Interface**: Offers a `Dialer` interface for constructing connections, which can be easily mocked for tests.
- **Observable Pattern**: Integrates the observable pattern, making it easier to react to chain events.

## Usage

### Basic Example

```go
// Creating a new EventsQueryClient with the default websocket dialer:
cometWebsocketURL := "ws://example.com"
evtClient := eventsquery.NewEventsQueryClient(cometWebsocketURL)

// Subscribing to a specific event:
observable, errCh := evtClient.EventsBytes(context.Background(), "your-query-string")
```

### Advanced Usage

[Further advanced examples can be added based on more sophisticated use cases, including setting custom dialers and handling observable outputs.]

### Configuration

- **WithDialer**: Configure the client to use a custom dialer for connections.

## API Reference

- `EventsQueryClient`: Main interface to query events. Methods include:
    - `EventsBytes(ctx, query)`: Returns an observable for chain events.
    - `Close()`: Close any existing connections and unsubscribe all observers.
- `Connection`: Interface representing a bidirectional message-passing connection.
- `Dialer`: Interface encapsulating the creation of connections.

For the complete API details, see the [godoc](https://pkg.go.dev/github.com/yourusername/pocket/pkg/client/events_query).

## Best Practices

- **Connection Handling**: Ensure to close the `EventsQueryClient` when done to free up resources and avoid potential leaks.
- **Error Handling**: Always check the error channel returned by `EventsBytes` for asynchronous errors during operation.

## FAQ

#### Why use `events_query` over directly using Gorilla WebSockets?

`events_query` abstracts many of the underlying details and provides a streamlined interface for subscribing to chain events. It also integrates the observable pattern and provides mockable interfaces for better testing.

#### How can I use a different connection mechanism other than WebSockets?

You can implement the `Dialer` and `Connection` interfaces and use the `WithDialer` configuration to provide your custom dialer.

## Contributing

If you're interested in improving the `events_query` package or adding new features, please start by discussing your ideas in the project's issues section. Check our main contributing guide for more details.

## Changelog

For detailed release notes, see the [CHANGELOG](../CHANGELOG.md).

## License

This package is released under the XYZ License. For more information, see the [LICENSE](../LICENSE) file at the root level.