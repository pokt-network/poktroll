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
	"sync/atomic"

	"go.uber.org/multierr"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/events/websocket"
	"github.com/pokt-network/poktroll/pkg/either"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

var _ client.EventsQueryClient = (*eventsQueryClient)(nil)

// TODO_TECHDEBT:
// - The cosmos-sdk CLI uses a cometbft RPC client with a `#Subscribe()` method for similar event subscription.
// - Consider replacing this custom websocket client with that approach.
// - References:
//   - https://github.com/cometbft/cometbft/blob/main/rpc/client/http/http.go#L110
//   - https://github.com/cometbft/cometbft/blob/main/rpc/client/http/http.go#L656
//   - https://github.com/cosmos/cosmos-sdk/blob/main/client/rpc/tx.go#L114
//   - https://github.com/pokt-network/poktroll/pull/64#discussion_r1372378241

// Implements client.EventsQueryClient.
type eventsQueryClient struct {
	logger polylog.Logger

	// cometWebsocketURL: Websocket URL for the cometbft node. Set in NewEventsQueryClient.
	cometWebsocketURL string

	// dialer: Creates the connection instance for cometbft node communication.
	dialer client.Dialer

	// eventsBytesAndConnsMu: Protects access to the eventsBytesAndConns map.
	eventsBytesAndConnsMu sync.RWMutex

	// eventsBytesInfo: Maps event subscription queries to their connection info.
	eventsBytesInfo map[string]*eventsBytesInfo

	// connId: Atomic counter for assigning unique connection IDs. Ensures thread safety.
	connId atomic.Uint64
}

// Holds information for managing event subscriptions.
type eventsBytesInfo struct {
	// eventsBytes: Observable notified about chain event messages matching the query.
	// Receives either an error or the event message bytes (either.Bytes).
	eventsBytes observable.Observable[either.Bytes]

	// eventsBzPublishCh: Channel to publish event bytes or errors to the eventsBytes observable.
	eventsBzPublishCh chan<- either.Bytes

	// query: Event subscription query for which this observable was created.
	query string

	// conn: Websocket connection to the cometbft node.
	conn client.Connection
	// connId: Unique identifier for the connection (for logging/debugging).
	connId uint64

	// ctx: Context managing the lifecycle of the eventsBytes observable.
	ctx context.Context
	// cancelCtx: Cancels the context to trigger resource cleanup.
	cancelCtx context.CancelFunc

	// isClosed: Indicates whether the connection is closed.
	// Use regular bool with mutex protection to reduce atomic contention.
	isClosed   bool
	isClosedMu sync.RWMutex
}

// NewEventsQueryClient returns a new events query client which is used to
// subscribe to onchain events matching the given query.
//
// Available options:
//   - WithDialer
func NewEventsQueryClient(cometWebsocketURL string, opts ...client.EventsQueryClientOption) client.EventsQueryClient {
	evtClient := &eventsQueryClient{
		cometWebsocketURL: cometWebsocketURL,
		eventsBytesInfo:   make(map[string]*eventsBytesInfo),
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
	if eventsBzConn := eqc.eventsBytesInfo[query]; eventsBzConn != nil {
		// If found it is returned.
		return eventsBzConn.eventsBytes, nil
	}

	// Check and close any existing connection for the given query.
	// This ensures that we do not have multiple subscriptions for the same query.
	// This is to prevent duplicate event notifications.
	if oldConn, ok := eqc.eventsBytesInfo[query]; ok {
		// If an old connection exists for the given query, close it.
		eqc.logger.Warn().Msgf(
			"♻️ Replacing existing event subscription for query %s and id %d with a new one",
			query,
			oldConn.connId,
		)

		// cancel the old context to stop the unsubscribe goroutine
		oldConn.cancelCtx()
	}

	// Otherwise, create a new event subscription for the given query.
	eventsBzConn, err := eqc.newEventsBytesAndConn(ctx, query)
	if err != nil {
		return nil, err
	}

	// Insert the new eventsBytes into the eventsBytesAndConns map.
	eqc.eventsBytesInfo[query] = eventsBzConn

	// Unsubscribe from the eventsBytes when the context is done.
	go eqc.goUnsubscribeOnDone(eventsBzConn)

	// Return the new eventsBytes observable for the given query.
	return eventsBzConn.eventsBytes, nil
}

// Close unsubscribes all observers from all event subscription observables.
func (eqc *eventsQueryClient) Close() {
	for _, eventsBzConn := range eqc.eventsBytesInfo {
		eqc.close(eventsBzConn)
	}
}

// close unsubscribes all observers from all event subscription observables.
func (eqc *eventsQueryClient) close(evtConn *eventsBytesInfo) {
	eqc.eventsBytesAndConnsMu.Lock()
	defer eqc.eventsBytesAndConnsMu.Unlock()

	// close handles the complete cleanup of an event subscription's resources to ensure proper termination:
	// 1. Sets the isClosed flag atomically to signal any goroutines to stop processing
	// 2. Closes the event publishing channel to terminate all downstream observers
	// 3. Closes the websocket connection to the CometBFT node
	// 4. Removes the subscription from the tracking map to prevent memory leaks
	//
	// This multi-step cleanup process ensures:
	// - Thread safety through mutex locking
	// - Prevention of goroutine leaks
	// - Avoidance of "send on closed channel" panics from late messages
	// - Proper resource cleanup of network connections
	// - Memory management by removing completed subscriptions
	if _, ok := eqc.eventsBytesInfo[evtConn.query]; ok {
		// mark the connection as closed to prevent late messages from being published
		// to the closed eventsBzPublishCh.
		evtConn.isClosedMu.Lock()
		evtConn.isClosed = true
		evtConn.isClosedMu.Unlock()

		// Unsubscribe all observers for the given query's eventsBzConn's observable and close its connection.
		close(evtConn.eventsBzPublishCh) // close the publish channel to stop the goroutine
		_ = evtConn.conn.Close()

		// Remove the eventsBytesAndConn for the given query.
		delete(eqc.eventsBytesInfo, evtConn.query)
		eqc.logger.Info().Msgf(
			"🗑️ Unsubscribed from events for query %s and id %d",
			evtConn.query,
			evtConn.connId,
		)
	} else {
		eqc.logger.Warn().Msgf(
			"⚠️ Failed to the event websocket connection for query %s: subscription with id %d not found. ❗ The connection has already been closed.",
			evtConn.query,
			evtConn.connId,
		)
	}
}

// newEventsBytesAndConn creates a new eventsBytes and connection for the given query.
func (eqc *eventsQueryClient) newEventsBytesAndConn(
	ctx context.Context,
	query string,
) (*eventsBytesInfo, error) {
	ctx, cancel := context.WithCancel(ctx)

	// Get a connection for the query.
	conn, err := eqc.openEventsBytesAndConn(ctx, query)
	if err != nil {
		cancel()
		return nil, err
	}

	// Construct an eventsBytes for the given query.
	eventsBzObservable, eventsBzPublishCh := channel.NewObservable[either.Bytes]()

	evtConn := &eventsBytesInfo{
		eventsBytes:       eventsBzObservable,
		eventsBzPublishCh: eventsBzPublishCh,

		query: query,

		conn:   conn,
		connId: eqc.connId.Add(1),

		ctx:       ctx,
		cancelCtx: cancel,
	}

	// Publish either events bytes or an error received from the connection to
	// the eventsBz observable.
	// NB: intentionally not retrying on error, leaving that to the caller.
	// (see: https://github.com/pokt-network/poktroll/pull/64#discussion_r1373826542)
	go eqc.goPublishEventsBz(evtConn)

	return evtConn, nil
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
	if err := conn.Send(req); err != nil {
		subscribeErr := ErrEventsSubscribe.Wrapf("%s", err)
		// assume the connection is bad
		closeErr := conn.Close()
		return nil, multierr.Combine(subscribeErr, closeErr)
	}

	// DEV_NOTE: The info log is done in a separate goroutine to prevent message ordering issues.
	// It ensures that "connection established" appears after any "connection closed" logs
	// from concurrent goPublishEventsBz goroutines.
	go eqc.logger.Info().Msgf(
		"🛜 Connection established to comet websocket endpoint %s",
		eqc.cometWebsocketURL,
	)

	return conn, nil
}

// goPublishEventsBz blocks on reading messages from a websocket connection.
// It is intended to be called from within a go routine.
func (eqc *eventsQueryClient) goPublishEventsBz(evtConn *eventsBytesInfo) {
	// Use defer to recover from potential panics due to race conditions
	defer func() {
		if r := recover(); r != nil {
			// Log the panic but don't crash the program
			if eqc.logger != nil {
				eqc.logger.Warn().Msgf("Recovered from panic in goPublishEventsBz: %v", r)
			}
		}
	}()

	// Read and handle messages from the websocket. This loop will exit when the
	// websocket connection is isClosed and/or returns an error.
	for {
		// Check context cancellation before each receive to avoid unnecessary work
		select {
		case <-evtConn.ctx.Done():
			return
		default:
		}

		// Check if connection is closed to avoid unnecessary reads in tight loop
		evtConn.isClosedMu.RLock()
		closed := evtConn.isClosed
		evtConn.isClosedMu.RUnlock()
		if closed {
			return
		}

		eventBz, err := evtConn.conn.Receive()

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
			if !errors.Is(evtConn.ctx.Err(), context.Canceled) {
				// Use a safe send helper to avoid panic
				eqc.safeSend(evtConn, either.Error[[]byte](err))
			}

			// Cancel the context to stop the unsubscribe goroutine
			// The cleanup logic is handled in the goUnsubscribeOnDone goroutine which
			// waits for the context to be done.
			evtConn.cancelCtx()

			return
		}

		// Use a safe send helper to avoid panic
		eqc.safeSend(evtConn, either.Success(eventBz))
	}
}

// safeSend safely sends a message to the eventsBzPublishCh channel, handling race conditions
func (eqc *eventsQueryClient) safeSend(evtConn *eventsBytesInfo, msg either.Bytes) {
	// Check if connection is closed before sending to avoid panic
	evtConn.isClosedMu.RLock()
	closed := evtConn.isClosed
	evtConn.isClosedMu.RUnlock()
	if closed {
		return
	}

	// Use select to avoid blocking if channel is full or closed
	select {
	case evtConn.eventsBzPublishCh <- msg:
	case <-evtConn.ctx.Done():
	default:
		// Channel is full or closed, drop the message
	}
}

// goUnsubscribeOnDone unsubscribes from the subscription when the context is done.
// It is intended to be called  in a goroutine.
func (eqc *eventsQueryClient) goUnsubscribeOnDone(
	evtConn *eventsBytesInfo,
) {
	// Wait for the context to be done.
	<-evtConn.ctx.Done()
	eqc.close(evtConn)
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
