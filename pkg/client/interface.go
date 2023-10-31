//go:generate mockgen -destination=../../internal/mocks/mockclient/query_client_mock.go -package=mockclient . Dialer,Connection

package client

import (
	"context"

	"pocket/pkg/either"
	"pocket/pkg/observable"
)

// TODO_CONSIDERATION: the cosmos-sdk CLI code seems to use a cometbft RPC client
// which includes a `#Subscribe()` method for a similar purpose. Perhaps we could
// replace this custom websocket client with that.
// (see: https://github.com/cometbft/cometbft/blob/main/rpc/client/http/http.go#L110)
// (see: https://github.com/cosmos/cosmos-sdk/blob/main/client/rpc/tx.go#L114)
//
// NOTE: a branch which attempts this is available at:
// https://github.com/pokt-network/poktroll/pull/74

// EventsBytesObservable is an observable which is notified with an either
// value which contains either an error or the event message bytes.
// TODO_HACK: The purpose of this type is to work around gomock's lack of
// support for generic types. For the same reason, this type cannot be an
// alias (i.e. EventsBytesObservable = observable.Observable[either.Bytes]).
type EventsBytesObservable observable.Observable[either.Bytes]

// EventsQueryClient is used to subscribe to chain event messages matching the given query,
type EventsQueryClient interface {
	// EventsBytes returns an observable which is notified about chain event messages
	// matching the given query. It receives an either value which contains either an
	// error or the event message bytes.
	EventsBytes(
		ctx context.Context,
		query string,
	) (EventsBytesObservable, error)
	// Close unsubscribes all observers of each active query's events bytes
	// observable and closes the connection.
	Close()
}

// Connection is a transport agnostic, bi-directional, message-passing interface.
type Connection interface {
	// Receive blocks until a message is received or an error occurs.
	Receive() (msg []byte, err error)
	// Send sends a message and may return a synchronous error.
	Send(msg []byte) error
	// Close closes the connection.
	Close() error
}

// Dialer encapsulates the construction of connections.
type Dialer interface {
	// DialContext constructs a connection to the given URL and returns it or
	// potentially a synchronous error.
	DialContext(ctx context.Context, urlStr string) (Connection, error)
}

// EventsQueryClientOption is an interface-wide type which can be implemented to use or modify the
// query client during construction. This would likely be done in an
// implementation-specific way; e.g. using a type assertion to assign to an
// implementation struct field(s).
type EventsQueryClientOption func(EventsQueryClient)
