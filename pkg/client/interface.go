//go:generate mockgen -destination=../../internal/mocks/mockclient/query_client_mock.go -package=mockclient . Dialer,Connection

package client

import (
	"context"
	"pocket/pkg/observable"
)

// TODO_CONSIDERATION: the cosmos-sdk CLI code seems to use a cometbft RPC client
// which includes a `#EventsObservable()` method for a similar purpose. Perhaps we could
// replace this custom websocket client with that.
// (see: https://github.com/cometbft/cometbft/blob/main/rpc/client/http/http.go#L110)
// (see: https://github.com/cosmos/cosmos-sdk/blob/main/client/rpc/tx.go#L114)

// QueryClient is used to subscribe to chain event messages matching the given query,
type QueryClient interface {
	EventsObservable(
		ctx context.Context,
		query string,
	) (observable.Observable[[]byte], chan error)
	// DISCUSS_THIS_COMMIT: do we care about returning an error?
	Close()
}

// CONSIDERATION: if the need arises in the future to support alternate and/or
// multiple transports, these interfaces could be repurposed and extended to
// that end. It would also likely involve adding implementations which adapt the
// underlying transport libraries to these interface.

type Connection interface {
	ReadEvent() (event []byte, err error)
	WriteJSON(any) error
	Close() error
}

type Dialer interface {
	DialContext(ctx context.Context, urlStr string) (Connection, error)
}

// MessageHandler is a function that handles a websocket chain-event subscription message.
type MessageHandler func(ctx context.Context, msg []byte) error

type Option func(QueryClient)
