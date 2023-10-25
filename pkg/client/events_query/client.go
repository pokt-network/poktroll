package eventsquery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"go.uber.org/multierr"

	"pocket/pkg/client"
	"pocket/pkg/either"
	"pocket/pkg/observable"
	"pocket/pkg/observable/channel"
)

const requestIdFmt = "request_%d"

var _ client.EventsQueryClient = &eventsQueryClient{}

// TODO_CONSIDERATION: the cosmos-sdk CLI code seems to use a cometbft RPC client
// which includes a `#EventsBytes()` method for a similar purpose. Perhaps we could
// replace this custom websocket client with that.
// (see: https://github.com/cometbft/cometbft/blob/main/rpc/client/http/http.go#L110)
// (see: https://github.com/cosmos/cosmos-sdk/blob/main/client/rpc/tx.go#L114)

// eventsQueryClient implements the EventsQueryClient interface.
type eventsQueryClient struct {
	// cometWebsocketURL is the websocket URL for the cometbft node. It is assigned
	// in NewEventsQueryClient.
	cometWebsocketURL string
	// nextRequestId is a *unique* ID intended to be monotonically incremented
	// and used to uniquely identify distinct RPC requests.
	// TODO_CONSIDERATION: Consider changing `nextRequestId` to a random entropy field
	nextRequestId uint64

	// dialer is resopnsible for createing the connection instance which
	// facilitates communication with the cometbft node via message passing.
	dialer client.Dialer
	// eventsBytesAndConnsMu protects the eventsBytesAndConns map.
	eventsBytesAndConnsMu sync.RWMutex
	// eventsBytesAndConns maps event subscription queries to their respective
	// eventsBytes observable, connection, and closed status.
	eventsBytesAndConns map[string]*eventsBytesAndConn
}

// eventsBytesAndConn is a struct which holds an eventsBytes observable & the
// corresponding connection which produces its inputs.
type eventsBytesAndConn struct {
	// eventsBytes is an observable which is notified about chain event messages
	// matching the given query. It receives an either.Either[[]byte] which is
	// either an error or the event message bytes.
	eventsBytes observable.Observable[either.Either[[]byte]]
	conn        client.Connection
	closed      bool
}

func NewEventsQueryClient(cometWebsocketURL string, opts ...client.EventsQueryClientOption) client.EventsQueryClient {
	evtClient := &eventsQueryClient{
		cometWebsocketURL:   cometWebsocketURL,
		eventsBytesAndConns: make(map[string]*eventsBytesAndConn),
	}

	for _, opt := range opts {
		opt(evtClient)
	}

	if evtClient.dialer == nil {
		// default to using the websocket dialer
		evtClient.dialer = NewWebsocketDialer()
	}

	return evtClient
}

// EventsBytes returns an eventsBytes observable which is notified about chain
// event messages matching the given query. It receives an either.Either[[]byte]
// which is either an error or the event message bytes.
// (see: https://pkg.go.dev/github.com/cometbft/cometbft/types#pkg-constants)
// (see: https://docs.cosmos.network/v0.47/core/events#subscribing-to-events)
func (eqc *eventsQueryClient) EventsBytes(
	ctx context.Context,
	query string,
) (client.EventsBytesObservable, error) {
	// Must (write) lock eventsBytesAndConnsMu so that we can safely check for
	// existing subscriptions to the given query or add a new eventsBytes to the
	// observableConns map.
	// The lock must be held for both checking and adding to prevent concurrent
	// calls to this function from racing.
	eqc.eventsBytesAndConnsMu.Lock()
	// Deferred (write) unlock.
	defer eqc.eventsBytesAndConnsMu.Unlock()

	// Check if an event subscription already exists for the given query.
	if eventsBzConn := eqc.eventsBytesAndConns[query]; eventsBzConn != nil {
		// If found it is returned.
		return eventsBzConn.eventsBytes, nil
	}

	// Otherwise, create a new event subscription for the given query.
	eventsBzConn, err := eqc.newEventsBytesAndConn(ctx, query)
	if err != nil {
		return nil, err
	}

	// Insert the new eventsBytes into the eventsBytesAndConns map.
	eqc.eventsBytesAndConns[query] = eventsBzConn

	// Unsubscribe from the eventsBytes when the context is done.
	go eqc.goUnsubscribeOnDone(ctx, query)

	// Return the new eventsBytes observable for the given query.
	return eventsBzConn.eventsBytes, nil
}

// Close unsubscribes all observers from all event subscription observables.
func (eqc *eventsQueryClient) Close() {
	eqc.close()
}

// close unsubscribes all observers from all event subscription observables.
func (eqc *eventsQueryClient) close() {
	eqc.eventsBytesAndConnsMu.Lock()
	defer eqc.eventsBytesAndConnsMu.Unlock()

	for query, obsvblConn := range eqc.eventsBytesAndConns {
		_ = obsvblConn.conn.Close()
		obsvblConn.eventsBytes.UnsubscribeAll()

		// remove closed eventsBytesAndConns
		delete(eqc.eventsBytesAndConns, query)
	}
}

// getNextRequestId increments and returns the JSON-RPC request ID which should
// be used for the next request. These IDs are expected to be unique (per request).
func (eqc *eventsQueryClient) getNextRequestId() string {
	eqc.nextRequestId++
	return fmt.Sprintf(requestIdFmt, eqc.nextRequestId)
}

// newEventwsBzAndConn creates a new eventsBytes and connection for the given query.
func (eqc *eventsQueryClient) newEventsBytesAndConn(
	ctx context.Context,
	query string,
) (*eventsBytesAndConn, error) {
	conn, err := eqc.openEventsBytesAndConn(ctx, query)
	if err != nil {
		return nil, err
	}

	// Construct an eventsBytes for the given query.
	eventsBzObservable, eventsBzPublishCh := channel.NewObservable[either.Either[[]byte]]()

	// TODO_INVESTIGATE: does this require retry on error?
	go eqc.goPublishEventsBz(ctx, conn, eventsBzPublishCh)

	return &eventsBytesAndConn{
		eventsBytes: eventsBzObservable,
		conn:        conn,
	}, nil
}

// openEventsBytesAndConn gets a connection using the configured dialer and sends
// an event subscription request on it, returning the connection.
func (eqc *eventsQueryClient) openEventsBytesAndConn(
	ctx context.Context,
	query string,
) (client.Connection, error) {
	// If no event subscription exists for the given query, create a new one.
	// Generate a new unique request ID.
	requestId := eqc.getNextRequestId()
	req, err := eventSubscriptionRequest(requestId, query)
	if err != nil {
		return nil, err
	}

	// Get a connection from the dialer.
	conn, err := eqc.dialer.DialContext(ctx, eqc.cometWebsocketURL)
	if err != nil {
		return nil, ErrDial.Wrapf("%s", err)
	}

	// Send the event subscription request on the connection.
	if err := conn.Send(req); err != nil {
		subscribeErr := ErrSubscribe.Wrapf("%s", err)
		// assume the connection is bad
		closeErr := conn.Close()
		return nil, multierr.Combine(subscribeErr, closeErr)
	}
	return conn, nil
}

// goPublishEventsBz blocks on reading messages from a websocket connection.
// It is intended to be called from within a go routine.
func (eqc *eventsQueryClient) goPublishEventsBz(
	ctx context.Context,
	conn client.Connection,
	eventsBzPublishCh chan<- either.Either[[]byte],
) {
	// Read and handle messages from the websocket. This loop will exit when the
	// websocket connection is closed and/or returns an error.
	for {
		event, err := conn.Receive()
		if err != nil {
			// TODO_CONSIDERATION: should we close the publish channel here too?

			// Stop this goroutine if there's an error.
			//
			// See gorilla websocket `Conn#NextReader()` docs:
			// | Applications must break out of the application's read loop when this method
			// | returns a non-nil error value. Errors returned from this method are
			// | permanent. Once this method returns a non-nil error, all subsequent calls to
			// | this method return the same error.

			// Only propagate error if it's not a context cancellation error.
			if !errors.Is(ctx.Err(), context.Canceled) {
				// Populate the error side (left) of the either and publish it.
				eventsBzPublishCh <- either.Error[[]byte](err)
			}

			eqc.close()
			return
		}

		// Populate the []byte side (right) of the either and publish it.
		eventsBzPublishCh <- either.Success(event)
	}
}

// goUnsubscribeOnDone unsubscribes from the subscription when the context is done.
// It is intended to be called  in a goroutine.
func (eqc *eventsQueryClient) goUnsubscribeOnDone(
	ctx context.Context,
	query string,
) {
	// wait for the context to be done
	<-ctx.Done()
	// only close the eventsBytes for the give query
	eqc.eventsBytesAndConnsMu.RLock()
	defer eqc.eventsBytesAndConnsMu.RUnlock()

	if toClose, ok := eqc.eventsBytesAndConns[query]; ok {
		toClose.eventsBytes.UnsubscribeAll()
	}
	for compareQuery, eventsBzConn := range eqc.eventsBytesAndConns {
		if query == compareQuery {
			eventsBzConn.eventsBytes.UnsubscribeAll()
			return
		}
	}
}

// eventSubscriptionRequest returns a JSON-RPC request for subscribing to events
// matching the given query.
// (see: https://github.com/cometbft/cometbft/blob/main/rpc/client/http/http.go#L110)
// (see: https://github.com/cosmos/cosmos-sdk/blob/main/client/rpc/tx.go#L114)
func eventSubscriptionRequest(requestId, query string) ([]byte, error) {
	requestJson := map[string]any{
		"jsonrpc": "2.0",
		"method":  "subscribe",
		"id":      requestId,
		"params": map[string]interface{}{
			"query": query,
		},
	}
	requestBz, err := json.Marshal(requestJson)
	if err != nil {
		return nil, err
	}
	return requestBz, nil
}
