package websockets

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	"github.com/pokt-network/pocket/pkg/observable"
	"github.com/pokt-network/pocket/pkg/polylog"
	"github.com/pokt-network/pocket/pkg/relayer"
)

const (
	// Time allowed (in seconds) to write a message to the peer.
	writeWaitSec = 10 * time.Second

	// Time allowed (in seconds) to wait for the next pong message to be received
	// from the peer before the connection closes.
	pongWaitSec = 30 * time.Second

	// Send pings to peer with this period.
	// Must be less than pongWaitSec.
	pingPeriodSec = (pongWaitSec * 9) / 10
)

// messageSource represents the source of a message in a bidirectional connection.
// It may be either `service_backend` or `gateway`.
type messageSource string

const (
	messageSourceServiceBackend messageSource = "service_backend"
	messageSourceGateway        messageSource = "gateway"
)

// message represents a message sent or received in a bidirectional connection.
// It may be a RelayRequest coming from the gateway or a raw message coming from
// the service backend.
type message struct {
	// data is the message payload
	data []byte

	// source may be either `gateay` or `service_backend`
	source messageSource

	// messageType is an int returned by the gorilla/websocket package.
	// It can be one of the following: Text: 1, Binary: 2, Close: 8, Ping: 9, Pong: 10 messages.
	// For more information, refer to: https://www.rfc-editor.org/rfc/rfc6455.html#section-11.8
	messageType int
}

// connection represents a websocket connection established between the relay miner and:
// - a gateway
// - a service backend
type connection struct {
	*websocket.Conn

	ctx    context.Context
	logger polylog.Logger

	// source is the source of the connection, it may be either `service_backend` or `gateway`.
	source messageSource

	// serviceID is the service ID of the service backend.
	serviceID string

	// msgChan is the channel where the messages are received by the relay miner.
	msgChan chan<- message

	// stopChan is the channel where the connection termination and errors are sent.
	stopChan chan<- error

	// stopObservable is the observable that notifies the connection to stop and errors.
	stopObservable observable.Observable[error]

	// isClosed is a flag that indicates whether the connection is closed.
	isClosed atomic.Bool
}

// connectServiceBackend establishes a websocket connection established by the
// relay miner client to the service backend endpoint.
func connectServiceBackend(serviceBackendUrl *url.URL, header http.Header) (*websocket.Conn, error) {
	// Create a new websocket dialer according to the service backend URL scheme.
	// If the scheme is `wss`, we need to create a new dialer with a custom TLS
	// configuration.
	// Otherwise, we use the default dialer.
	var dialer *websocket.Dialer
	switch serviceBackendUrl.Scheme {
	case "wss":
		dialer = &websocket.Dialer{TLSClientConfig: &tls.Config{}}
	default:
		dialer = websocket.DefaultDialer
	}

	conn, _, err := dialer.Dial(serviceBackendUrl.String(), header)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// newConnection manages the lifecycle of a websocket connection.
func newConnection(
	ctx context.Context,
	conn *websocket.Conn,
	logger polylog.Logger,
	source messageSource,
	serviceID string,
	msgChan chan<- message,
	stopChan chan<- error,
	stopObservable observable.Observable[error],
) *connection {
	connectionLogger := logger.With(
		"connection_source", string(source),
		"service_id", serviceID,
	)

	c := &connection{
		ctx:            ctx,
		Conn:           conn,
		logger:         connectionLogger,
		source:         source,
		serviceID:      serviceID,
		msgChan:        msgChan,
		stopChan:       stopChan,
		stopObservable: stopObservable,
	}

	c.isClosed.Store(false)

	// Start the connection's message and ping loops.
	go c.connLoop()
	go c.pingLoop()

	// Schedule the cleanup function to run when stopObservable emits a value.
	go c.cleanup()

	return c
}

// connLoop reads messages from the websocket connection and sends them to the
// bridge's message channel.
func (c *connection) connLoop() {
	for {
		// Read the next message from the websocket connection and forward it to the
		// message channel.
		messageType, msg, err := c.ReadMessage()

		// Stop the loop if the connection is closing.
		if c.isClosed.Load() {
			return
		}

		if err != nil {
			c.handleError(err)
			return
		}

		c.msgChan <- message{
			data:        msg,
			source:      c.source,
			messageType: messageType,
		}
	}
}

// pingLoop sends keep-alive ping messages to the peer at regular intervals.
// If the peer does not respond with a pong message within the allowed time,
// the connection is closed.
func (c *connection) pingLoop() {
	logger := c.logger.With("connection_context", "pingLoop")

	ticker := time.NewTicker(pingPeriodSec)
	defer ticker.Stop()

	// Set the read deadline for the first pong message.
	if err := c.SetReadDeadline(time.Now().Add(pongWaitSec)); err != nil {
		logger.Error().Err(err).Msg("failed to set initial read deadline")
		c.handleError(ErrWebsocketsConnection.Wrapf("failed to set initial read deadline: %v", err))
		return
	}

	// Each time a pong message is received, set the read deadline for the next one.
	c.SetPongHandler(func(string) error {
		if err := c.SetReadDeadline(time.Now().Add(pongWaitSec)); err != nil {
			logger.Error().Err(err).Msg("failed to set pong handler read deadline")
			c.handleError(ErrWebsocketsConnection.Wrapf("failed to set pong handler read deadline: %v", err))
			return err
		}

		return nil
	})

	for {
		select {
		case <-c.stopObservable.Subscribe(c.ctx).Ch():
			return

		// Send a ping message to the peer at regular intervals.
		case <-ticker.C:
			if err := c.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWaitSec)); err != nil {
				logger.Error().Err(err).Msg("failed to send ping to connection")
				c.handleError(ErrWebsocketsConnection.Wrapf("failed to send ping to connection: %v", err))
				return
			}
		}
	}
}

// handleError logs the error and sends it to the stop channel.
func (c *connection) handleError(err error) {
	logger := c.logger.With("connection_context", "handleError")

	switch {
	case websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway):
		logger.Info().Err(err).Msg("connection closed by peer")
	case websocket.IsCloseError(err, websocket.CloseNoStatusReceived, websocket.CloseAbnormalClosure):
		logger.Warn().Err(err).Msg("connection going away")
	default:
		relayer.RelaysErrorsTotal.With("service_id", c.serviceID).Add(1)
		logger.Error().Err(err).Msg("connection closed unexpectedly")
	}

	c.stopChan <- err
}

// cleanup closes the websocket connection and sends a close message to the peer.
func (c *connection) cleanup() {
	logger := c.logger.With("connection_context", "cleanup")

	// Wait for the stop observable to emit a value before cleaning up the connection.
	err := <-c.stopObservable.Subscribe(c.ctx).Ch()

	if !c.isClosed.CompareAndSwap(false, true) {
		logger.Info().Msg("connection already closed")
	}

	logger.Info().Msg("connection closing, cleaning up")

	// Determine the close code and text.
	closeCode := websocket.CloseNormalClosure
	closeText := "connection closed"
	if err != nil {
		closeText = err.Error()
		// If it's a websocket close error, use its code and text.
		if e, ok := err.(*websocket.CloseError); ok {
			closeCode = e.Code
			closeText = e.Text
		}
	}

	// Format and send the close message.
	closeMsg := websocket.FormatCloseMessage(closeCode, closeText)
	deadline := time.Now().Add(writeWaitSec)
	if err := c.WriteControl(websocket.CloseMessage, closeMsg, deadline); err != nil {
		logger.Error().Err(err).Msg("failed to send close message")
	}

	// Wait for the control message to be received by the peer.
	time.Sleep(writeWaitSec)
	if err := c.Close(); err != nil {
		logger.Error().Err(err).Msg("failed to close connection")
	}
	logger.Info().Msg("connection closed")
}
