package events

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sync"

	"go.uber.org/multierr"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/events/websocket"
	"github.com/pokt-network/poktroll/pkg/either"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
)

var _ client.EventsQueryClient = (*eventsQueryClient)(nil)

// TODO_TECHDEBT: the cosmos-sdk CLI code seems to use a cometbft RPC client
// which includes a `#Subscribe()` method for a similar purpose. Perhaps we could
// replace this custom websocket client with that.
// See:
// - https://github.com/cometbft/cometbft/blob/main/rpc/client/http/http.go#L110
// - https://github.com/cometbft/cometbft/blob/main/rpc/client/http/http.go#L656
// - https://github.com/cosmos/cosmos-sdk/blob/main/client/rpc/tx.go#L114
// - https://github.com/pokt-network/poktroll/pull/64#discussion_r1372378241

// eventsQueryClient implements the EventsQueryClient interface.
type eventsQueryClient struct {
	// cometWebsocketURL is the websocket URL for the cometbft node. It is assigned
	// in NewEventsQueryClient.
	cometWebsocketURL string
	// dialer is responsible for creating the connection instance which
	// facilitates communication with the cometbft node via message passing.
	dialer client.Dialer
	// eventsBytesAndConnsMu protects the eventsBytesAndConns map.
	eventsBytesAndConnsMu sync.RWMutex
	// eventsBytesAndConns maps event subscription queries to their respective
	// eventsBytes observable, connection, and isClosed status.
	eventsBytesAndConns map[string]*eventsBytesAndConn
}

// eventsBytesAndConn is a struct which holds an eventsBytes observable & the
// corresponding connection which produces its inputs.
type eventsBytesAndConn struct {
	// eventsBytes is an observable which is notified about chain event messages
	// matching the given query. It receives an either.Bytes which is
	// either an error or the event message bytes.
	eventsBytes observable.Observable[either.Bytes]
	conn        client.Connection
}

// Close unsubscribes all observers of eventsBytesAndConn's observable and also
// closes its connection.
func (ebc *eventsBytesAndConn) Close() {
	ebc.eventsBytes.UnsubscribeAll()
	_ = ebc.conn.Close()
}

// NewEventsQueryClient returns a new events query client which is used to
// subscribe to on-chain events matching the given query.
//
// Available options:
//   - WithDialer
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
		evtClient.dialer = websocket.NewWebsocketDialer()
	}

	return evtClient
}

// EventsBytes returns an eventsBytes observable which is notified about chain
// event messages matching the given query. It receives an either.Bytes
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

	for query, eventsBzConn := range eqc.eventsBytesAndConns {
		// Unsubscribe all observers of the eventsBzConn observable and close the
		// connection for the given query.
		eventsBzConn.Close()
		// remove isClosed eventsBytesAndConns
		delete(eqc.eventsBytesAndConns, query)
	}
}

// newEventwsBzAndConn creates a new eventsBytes and connection for the given query.
func (eqc *eventsQueryClient) newEventsBytesAndConn(
	ctx context.Context,
	query string,
) (*eventsBytesAndConn, error) {
	// Get a connection for the query.
	conn, err := eqc.openEventsBytesAndConn(ctx, query)
	if err != nil {
		return nil, err
	}

	// Construct an eventsBytes for the given query.
	eventsBzObservable, eventsBzPublishCh := channel.NewObservable[either.Bytes]()

	// Publish either events bytes or an error received from the connection to
	// the eventsBz observable.
	// NB: intentionally not retrying on error, leaving that to the caller.
	// (see: https://github.com/pokt-network/poktroll/pull/64#discussion_r1373826542)
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
	// Get a request for subscribing to events matching the given query.
	req, err := eqc.eventSubscriptionRequest(query)
	if err != nil {
		return nil, err
	}

	// Get a connection from the dialer.
	conn, err := eqc.dialer.DialContext(ctx, eqc.cometWebsocketURL)
	if err != nil {
		return nil, ErrEventsDial.Wrapf("%s", err)
	}

	// Send the event subscription request on the connection.
	if err = conn.Send(req); err != nil {
		subscribeErr := ErrEventsSubscribe.Wrapf("%s", err)
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
	eventsBzPublishCh chan<- either.Bytes,
) {
	// Read and handle messages from the websocket. This loop will exit when the
	// websocket connection is isClosed and/or returns an error.
	for {
		eventBz, err := conn.Receive()
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
		eventsBzPublishCh <- either.Success(eventBz)
	}
}

// goUnsubscribeOnDone unsubscribes from the subscription when the context is done.
// It is intended to be called  in a goroutine.
func (eqc *eventsQueryClient) goUnsubscribeOnDone(
	ctx context.Context,
	query string,
) {
	// Wait for the context to be done.
	<-ctx.Done()
	// Only close the eventsBytes for the given query.
	eqc.eventsBytesAndConnsMu.Lock()
	defer eqc.eventsBytesAndConnsMu.Unlock()

	if eventsBzConn, ok := eqc.eventsBytesAndConns[query]; ok {
		// Unsubscribe all observers of the given query's eventsBzConn's observable
		// and close its connection.
		eventsBzConn.Close()
		// Remove the eventsBytesAndConn for the given query.
		delete(eqc.eventsBytesAndConns, query)
	}
}

// eventSubscriptionRequest returns a JSON-RPC request for subscribing to events
// matching the given query. The request is serialized as JSON to a byte slice.
// (see: https://github.com/cometbft/cometbft/blob/main/rpc/client/http/http.go#L110)
// (see: https://github.com/cosmos/cosmos-sdk/blob/main/client/rpc/tx.go#L114)
func (eqc *eventsQueryClient) eventSubscriptionRequest(query string) ([]byte, error) {
	requestJson := map[string]any{
		"jsonrpc": "2.0",
		"method":  "subscribe",
		"id":      randRequestId(),
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

// randRequestId returns a random 8 byte, base64 request ID which is intended
// for in JSON-RPC requests to uniquely identify distinct RPC requests.
// These request IDs only need to be unique to the extent that they are useful
// to this client for identifying distinct RPC requests. Their size and keyspace
// are arbitrary.
func randRequestId() string {
	requestIdBz := make([]byte, 8) // 8 bytes = 64 bits = uint64
	if _, err := rand.Read(requestIdBz); err != nil {
		panic(fmt.Sprintf(
			"failed to generate random request ID: %s", err,
		))
	}
	return base64.StdEncoding.EncodeToString(requestIdBz)
}

// RPCToWebsocketURL converts the provided URL into a websocket URL string that can
// be used to subscribe to onchain events and query the chain via a client
// context or send transactions via a tx client context.
func RPCToWebsocketURL(hostUrl *url.URL) string {
	switch hostUrl.Scheme {
	case "http":
		fallthrough
	case "ws":
		fallthrough
	case "tcp":
		return fmt.Sprintf("ws://%s/websocket", hostUrl.Host)
	default:
		return fmt.Sprintf("wss://%s/websocket", hostUrl.Host)
	}
}
