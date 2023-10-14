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

// EventsQueryClient is used to subscribe to chain event messages matching the given query,
type EventsQueryClient interface {
	EventsBytes(
		ctx context.Context,
		query string,
	) (EventsBytesObservable, error)
	//EventsBytes(
	//	ctx context.Context,
	//	query string,
	//) (observable.Observable[either.Either[[]byte]], error)
	Close()
}

type EventsBytesObservable observable.Observable[either.Either[[]byte]]

// Connection is a transport agnostic, bi-directional, message-passing interface.
type Connection interface {
	Receive() (msg []byte, err error)
	Send(msg []byte) error
	Close() error
}

// Dialer encapsulates the construction of connections.
type Dialer interface {
	DialContext(ctx context.Context, urlStr string) (Connection, error)
}

// EventsQueryClientOption is an interface-wide type which can be implemented to use or modify the
// query client during construction. This would likely be done in an
// implementation-specific way; e.g. using a type assertion to assign to an
// implementation struct field(s).
type EventsQueryClientOption func(EventsQueryClient)
