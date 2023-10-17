package query

import (
	"context"
	"errors"
	"fmt"
	"log"
	"pocket/pkg/observable"
	"pocket/pkg/observable/channel"
	"sync"

	"go.uber.org/multierr"

	"pocket/pkg/client"
)

// TODO_CONSIDERATION: the cosmos-sdk CLI code seems to use a cometbft RPC client
// which includes a `#EventsObservable()` method for a similar purpose. Perhaps we could
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

	dialer   client.Dialer
	eventsMu sync.Mutex
	events   map[string]*eventStat
}

func NewQueryClient(cometWebsocketURL string, opts ...client.Option) client.QueryClient {
	qClient := &queryClient{
		cometWebsocketURL: cometWebsocketURL,
		events:            make(map[string]*eventStat),
	}

	for _, opt := range opts {
		opt(qClient)
	}

	if qClient.dialer == nil {
		// default to using the websocket dialer
		qClient.dialer = NewWebsocketDialer()
	}

	return qClient
}

func WithDialer(dialer client.Dialer) client.Option {
	return func(qClient client.QueryClient) {
		qClient.(*queryClient).dialer = dialer
	}
}

// TODO_THIS_COMMIT: move
type eventStat struct {
	sync.Mutex
	observable observable.Observable[[]byte]
	conn       client.Connection
	errCh      chan error
}

// SubscribeWithQuery subscribes to chain event messages matching the given query,
// via a websocket connection.
// (see: https://pkg.go.dev/github.com/cometbft/cometbft/types#pkg-constants)
// (see: https://docs.cosmos.network/v0.47/core/events#subscribing-to-events)
func (qClient *queryClient) EventsObservable(
	ctx context.Context,
	query string,
) (observable.Observable[[]byte], chan error) {
	events, ok := qClient.events[query]
	if ok {
		return events.observable, events.errCh
	}

	errCh := make(chan error, 1)
	conn, err := qClient.dialer.DialContext(ctx, qClient.cometWebsocketURL)
	if err != nil {
		errCh <- fmt.Errorf("failed to connect to websocket: %w", err)
		return nil, errCh
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
		// assume the connection is bad
		closeErr := conn.Close()
		errCh <- multierr.Combine(subscribeErr, closeErr)
		return nil, errCh
	}

	eventsObservable, eventsProducer := channel.NewObservable[[]byte]()
	qClient.events[query] = &eventStat{
		observable: eventsObservable,
		conn:       conn,
		errCh:      errCh,
	}

	go func() {
		if err := qClient.goProduceEvents(conn, eventsProducer); err != nil {
			// only propagate error if it's not a context cancellation error
			if !errors.Is(ctx.Err(), context.Canceled) {
				// TODO_THIS_COMMIT: refactor to cosmos-sdk error
				errCh <- fmt.Errorf("error listening on connection: %w", err)
				qClient.close()
				return
			}
		}
	}()

	go func() {
		<-ctx.Done()
		log.Println("closing websocket")
		qClient.close()
		fmt.Println("done closing")
	}()

	return eventsObservable, errCh
}

func (qClient *queryClient) Close() {
	qClient.close()
}

func (qClient *queryClient) close() {
	qClient.eventsMu.Lock()
	defer qClient.eventsMu.Unlock()

	for _, event := range qClient.events {
		fmt.Println("closing conn.observable...")
		_ = event.conn.Close()
		fmt.Println("... conn.observable closed")
		fmt.Println("closing event.observable...")
		event.observable.Close()
		fmt.Println("... event.observable closed")
	}
}

// goProduceEvents blocks on reading messages from a websocket connection.
// IMPORTANT: it is intended to be called from within a go routine.
func (qClient *queryClient) goProduceEvents(
	conn client.Connection,
	eventsProducer chan<- []byte,
) error {
	// read and handle messages from the websocket. This loop will exit when the
	// websocket connection is closed and/or returns an error.
	//var eventCounter int
	for {
		//fmt.Printf("event-%d\n", eventCounter)
		event, err := conn.ReadEvent()
		if err != nil {
			// TODO_THIS_COMMIT: close producer?

			// Stop this goroutine if there's an error.
			//
			// See gorilla websocket `Conn#NextReader()` docs:
			// | Applications must break out of the application's read loop when this method
			// | returns a non-nil error value. Errors returned from this method are
			// | permanent. Once this method returns a non-nil error, all subsequent calls to
			// | this method return the same error.
			return err
		}
		//eventCounter++

		eventsProducer <- event
	}
}

// getNextRequestId increments and returns the JSON-RPC request ID which should
// be used for the next request. These IDs are expected to be unique (per request).
func (qClient *queryClient) getNextRequestId() uint64 {
	qClient.nextRequestId++
	return qClient.nextRequestId
}
