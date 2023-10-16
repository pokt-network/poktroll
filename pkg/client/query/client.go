package query

import (
	"context"
	"errors"
	"fmt"
	"log"

	"go.uber.org/multierr"

	"pocket/pkg/client"
)

// TODO_CONSIDERATION: the cosmos-sdk CLI code seems to use a cometbft RPC client
// which includes a `#Subscribe()` method for a similar purpose. Perhaps we could
// replace this custom websocket client with that.
// (see: https://github.com/cometbft/cometbft/blob/main/rpc/client/http/http.go#L110)
// (see: https://github.com/cosmos/cosmos-sdk/blob/main/client/rpc/tx.go#L114)

// queryClient implements the QueryClient interface.
type queryClient struct {
	// cometWebsocketURL is the websocket URL for the cometbft node. It is assigned
	// in NewQueryClient.
	cometWebsocketURL string
	// nextRequestId is a *unique* ID intended to be monotonically incremented
	// and used to uniquely identify distinct RPC requests.
	// TODO_CONSIDERATION: Consider changing `nextRequestId` to a random entropy field
	nextRequestId uint64

	dialer client.Dialer
}

func NewQueryClient(cometWebsocketURL string, opts ...client.Option) client.QueryClient {
	qClient := &queryClient{cometWebsocketURL: cometWebsocketURL}

	for _, opt := range opts {
		opt(qClient)
	}

	return qClient
}

func WithDialer(dialer client.Dialer) client.Option {
	return func(qClient client.QueryClient) {
		qClient.(*queryClient).dialer = dialer
	}
}

// SubscribeWithQuery subscribes to chain event messages matching the given query,
// via a websocket connection.
// (see: https://pkg.go.dev/github.com/cometbft/cometbft/types#pkg-constants)
// (see: https://docs.cosmos.network/v0.47/core/events#subscribing-to-events)
func (qClient *queryClient) Subscribe(ctx context.Context, query string, msgHandler client.MessageHandler) chan error {
	errCh := make(chan error, 1)

	conn, err := qClient.dialer.DialContext(ctx, qClient.cometWebsocketURL)
	if err != nil {
		errCh <- fmt.Errorf("failed to connect to websocket: %w", err)
		return errCh
	}

	// TODO_DISCUSS: Should we replace `requestId` with just
	requestId := qClient.getNextRequestId()
	if err := conn.WriteJSON(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "subscribe",
		"id":      requestId,
		"params": map[string]interface{}{
			"query": query,
		},
	}); err != nil {
		// TODO_THIS_COMMIT: refactor to cosmos-sdk error
		subscribeErr := fmt.Errorf("failed to write subscribe request to websocket: %w", err)
		closeErr := conn.Close()
		errCh <- multierr.Combine(subscribeErr, closeErr)
		return errCh
	}

	go func() {
		if err := qClient.goListen(ctx, conn, msgHandler); err != nil {
			// only propagate error if it's not a context cancellation error
			if !errors.Is(ctx.Err(), context.Canceled) {
				// TODO_THIS_COMMIT: refactor to cosmos-sdk error
				errCh <- fmt.Errorf("error listening on connection: %w", err)
			}
		}
	}()

	go func() {
		<-ctx.Done()
		log.Println("closing websocket")
		_ = conn.Close()
	}()

	return errCh
}

// goListen blocks on reading messages from a websocket connection.
// IMPORTANT: it is intended to be called from within a go routine.
func (qClient *queryClient) goListen(
	ctx context.Context,
	conn client.Connection,
	msgHandler client.MessageHandler,
) error {
	// read and handle messages from the websocket. This loop will exit when the
	// websocket connection is closed and/or returns an error.
	for {
		msg, err := conn.ReadMessage()
		if err != nil {
			// Stop this goroutine if there's an error.
			//
			// See gorilla websocket `Conn#NextReader()` docs:
			// | Applications must break out of the application's read loop when this method
			// | returns a non-nil error value. Errors returned from this method are
			// | permanent. Once this method returns a non-nil error, all subsequent calls to
			// | this method return the same error.
			return err
		}

		if err := msgHandler(ctx, msg); err != nil {
			log.Printf("failed to handle msg: %s\n", err)
			continue
		}
	}
}

// getNextRequestId increments and returns the JSON-RPC request ID which should
// be used for the next request. These IDs are expected to be unique (per request).
func (qClient *queryClient) getNextRequestId() uint64 {
	qClient.nextRequestId++
	return qClient.nextRequestId
}
