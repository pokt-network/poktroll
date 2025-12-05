package relayer

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	"github.com/pokt-network/poktroll/pkg/ha/logging"
	"github.com/pokt-network/poktroll/pkg/ha/transport"
	"github.com/pokt-network/poktroll/pkg/polylog"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
)

const (
	// wsWriteWait is the time allowed to write a message to the peer.
	wsWriteWait = 10 * time.Second

	// wsPongWait is the time allowed to wait for the next pong message.
	wsPongWait = 30 * time.Second

	// wsPingPeriod is the send pings to peer with this period. Must be less than pongWait.
	wsPingPeriod = (wsPongWait * 9) / 10
)

// wsMessageSource represents the source of a WebSocket message.
type wsMessageSource string

const (
	wsMessageSourceBackend wsMessageSource = "backend"
	wsMessageSourceGateway wsMessageSource = "gateway"
)

// wsMessage represents a message in the WebSocket bridge.
type wsMessage struct {
	data        []byte
	source      wsMessageSource
	messageType int
}

// WebSocketBridge handles bidirectional WebSocket communication between
// a gateway client and a backend service.
type WebSocketBridge struct {
	logger         polylog.Logger
	gatewayConn    *websocket.Conn
	backendConn    *websocket.Conn
	relayProcessor RelayProcessor
	publisher      transport.MinedRelayPublisher
	responseSigner *ResponseSigner

	// Message channel for bridge communication
	msgChan chan wsMessage

	// Track latest request/response for pairing
	latestRequest  *servicetypes.RelayRequest
	latestResponse *servicetypes.RelayResponse
	latestMu       sync.RWMutex

	// Service and supplier info
	serviceID       string
	supplierAddress string
	arrivalHeight   int64

	// Relay counting for billing
	relayCount atomic.Uint64

	// Lifecycle
	ctx      context.Context
	cancelFn context.CancelFunc
	closed   atomic.Bool
	wg       sync.WaitGroup
}

// WebSocketUpgrader upgrades HTTP connections to WebSocket.
var WebSocketUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	// Accept connections from any origin for cross-origin support
	CheckOrigin: func(r *http.Request) bool { return true },
}

// NewWebSocketBridge creates a new WebSocket bridge.
func NewWebSocketBridge(
	logger polylog.Logger,
	gatewayConn *websocket.Conn,
	backendURL string,
	serviceID string,
	supplierAddress string,
	arrivalHeight int64,
	relayProcessor RelayProcessor,
	publisher transport.MinedRelayPublisher,
	responseSigner *ResponseSigner,
	headers http.Header,
) (*WebSocketBridge, error) {
	ctx, cancelFn := context.WithCancel(context.Background())

	// Connect to backend WebSocket
	backendConn, err := connectWebSocketBackend(backendURL, headers)
	if err != nil {
		cancelFn()
		return nil, err
	}

	bridge := &WebSocketBridge{
		logger:          logger.With(logging.FieldComponent, logging.ComponentWebsocketBridge, logging.FieldServiceID, serviceID),
		gatewayConn:     gatewayConn,
		backendConn:     backendConn,
		relayProcessor:  relayProcessor,
		publisher:       publisher,
		responseSigner:  responseSigner,
		msgChan:         make(chan wsMessage, 100),
		serviceID:       serviceID,
		supplierAddress: supplierAddress,
		arrivalHeight:   arrivalHeight,
		ctx:             ctx,
		cancelFn:        cancelFn,
	}

	// Track connection
	wsConnectionsActive.WithLabelValues(serviceID).Inc()
	wsConnectionsTotal.WithLabelValues(serviceID).Inc()

	return bridge, nil
}

// connectWebSocketBackend establishes a WebSocket connection to the backend.
func connectWebSocketBackend(backendURL string, headers http.Header) (*websocket.Conn, error) {
	parsedURL, err := url.Parse(backendURL)
	if err != nil {
		return nil, err
	}

	// Use TLS for wss:// scheme
	var dialer *websocket.Dialer
	if parsedURL.Scheme == "wss" {
		dialer = &websocket.Dialer{TLSClientConfig: &tls.Config{}}
	} else {
		dialer = websocket.DefaultDialer
	}

	conn, _, err := dialer.Dial(backendURL, headers)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// Run starts the WebSocket bridge message loop.
// This is a blocking call that runs until the bridge is closed.
func (b *WebSocketBridge) Run() {
	// Start connection read loops
	b.wg.Add(2)
	go b.readLoop(b.gatewayConn, wsMessageSourceGateway)
	go b.readLoop(b.backendConn, wsMessageSourceBackend)

	// Start ping loops for keep-alive
	b.wg.Add(2)
	go b.pingLoop(b.gatewayConn, "gateway")
	go b.pingLoop(b.backendConn, "backend")

	// Main message processing loop
	b.messageLoop()

	// Wait for all goroutines to finish
	b.wg.Wait()

	b.logger.Info().Msg("websocket bridge stopped")
}

// readLoop reads messages from a WebSocket connection.
func (b *WebSocketBridge) readLoop(conn *websocket.Conn, source wsMessageSource) {
	defer b.wg.Done()

	for {
		if b.closed.Load() {
			return
		}

		messageType, data, err := conn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				b.logger.Warn().
					Err(err).
					Str(logging.FieldSource, string(source)).
					Msg("websocket read error")
			}
			b.Close()
			return
		}

		select {
		case <-b.ctx.Done():
			return
		case b.msgChan <- wsMessage{data: data, source: source, messageType: messageType}:
		}
	}
}

// pingLoop sends periodic ping messages to keep the connection alive.
func (b *WebSocketBridge) pingLoop(conn *websocket.Conn, name string) {
	defer b.wg.Done()

	ticker := time.NewTicker(wsPingPeriod)
	defer ticker.Stop()

	// Set initial read deadline
	if err := conn.SetReadDeadline(time.Now().Add(wsPongWait)); err != nil {
		b.logger.Debug().Err(err).Str("connection", name).Msg("failed to set initial read deadline")
	}

	// Reset deadline on pong
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(wsPongWait))
	})

	for {
		select {
		case <-b.ctx.Done():
			return
		case <-ticker.C:
			if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(wsWriteWait)); err != nil {
				b.logger.Debug().
					Err(err).
					Str("connection", name).
					Msg("failed to send ping")
				b.Close()
				return
			}
		}
	}
}

// messageLoop processes messages from both connections.
func (b *WebSocketBridge) messageLoop() {
	for {
		select {
		case <-b.ctx.Done():
			return
		case msg := <-b.msgChan:
			switch msg.source {
			case wsMessageSourceGateway:
				b.handleGatewayMessage(msg)
			case wsMessageSourceBackend:
				b.handleBackendMessage(msg)
			}
		}
	}
}

// handleGatewayMessage handles messages from the gateway.
func (b *WebSocketBridge) handleGatewayMessage(msg wsMessage) {
	wsMessagesForwarded.WithLabelValues(b.serviceID, "gateway_to_backend").Inc()

	// Try to parse as RelayRequest
	relayReq := &servicetypes.RelayRequest{}
	if err := relayReq.Unmarshal(msg.data); err != nil {
		// Not a valid RelayRequest - forward raw data to backend
		b.forwardToBackend(msg)
		return
	}

	// Store latest request for pairing
	b.setLatestRequest(relayReq)

	// Forward payload to backend
	if err := b.backendConn.WriteMessage(msg.messageType, relayReq.Payload); err != nil {
		b.logger.Warn().Err(err).Msg("failed to forward to backend")
		b.Close()
		return
	}
}

// handleBackendMessage handles messages from the backend.
// Each backend message is billed as part of a relay.
func (b *WebSocketBridge) handleBackendMessage(msg wsMessage) {
	wsMessagesForwarded.WithLabelValues(b.serviceID, "backend_to_gateway").Inc()

	latestReq := b.getLatestRequest()
	if latestReq == nil {
		// No request yet - just forward raw data
		if err := b.gatewayConn.WriteMessage(msg.messageType, msg.data); err != nil {
			b.logger.Warn().Err(err).Msg("failed to forward to gateway")
			b.Close()
		}
		return
	}

	// Build and sign RelayResponse
	var respBytes []byte
	var relayResp *servicetypes.RelayResponse

	if b.responseSigner != nil {
		// Build signed response using the signer (like HTTP handler does)
		var signErr error
		relayResp, respBytes, signErr = b.responseSigner.BuildAndSignRelayResponseFromBody(
			latestReq,
			msg.data,
			nil, // No HTTP headers for WebSocket
			200, // Assume OK status for WebSocket messages
		)
		if signErr != nil {
			b.logger.Warn().Err(signErr).Msg("failed to sign websocket response")
			// Fall back to unsigned
			relayResp = &servicetypes.RelayResponse{
				Meta: servicetypes.RelayResponseMetadata{
					SessionHeader: latestReq.Meta.SessionHeader,
				},
				Payload: msg.data,
			}
			respBytes, _ = relayResp.Marshal()
		}
	} else {
		// No signer - create unsigned response
		relayResp = &servicetypes.RelayResponse{
			Meta: servicetypes.RelayResponseMetadata{
				SessionHeader: latestReq.Meta.SessionHeader,
			},
			Payload: msg.data,
		}
		var marshalErr error
		respBytes, marshalErr = relayResp.Marshal()
		if marshalErr != nil {
			b.logger.Warn().Err(marshalErr).Msg("failed to marshal response")
			respBytes = msg.data
		}
	}

	// Store latest response for relay emission
	b.setLatestResponse(relayResp)

	// Forward signed response to gateway
	if err := b.gatewayConn.WriteMessage(msg.messageType, respBytes); err != nil {
		b.logger.Warn().Err(err).Msg("failed to forward to gateway")
		b.Close()
		return
	}

	// Emit relay for this request/response pair
	b.emitRelay(latestReq, relayResp, msg.data)

	// Clear the request so next backend message waits for a new request
	b.clearLatestRequest()
}

// forwardToBackend forwards a raw message to the backend.
func (b *WebSocketBridge) forwardToBackend(msg wsMessage) {
	if err := b.backendConn.WriteMessage(msg.messageType, msg.data); err != nil {
		b.logger.Warn().Err(err).Msg("failed to forward raw message to backend")
		b.Close()
	}
}

// emitRelay creates and publishes a mined relay for a request/response pair.
// This is the billing mechanism - each req/resp pair becomes a relay.
func (b *WebSocketBridge) emitRelay(req *servicetypes.RelayRequest, resp *servicetypes.RelayResponse, respPayload []byte) {
	if b.publisher == nil {
		return
	}

	// Increment relay count for this connection
	count := b.relayCount.Add(1)

	// Get supplier address from request metadata or fallback to bridge config
	supplierAddr := b.supplierAddress
	if req.Meta.SupplierOperatorAddress != "" {
		supplierAddr = req.Meta.SupplierOperatorAddress
	}

	if supplierAddr == "" {
		b.logger.Warn().Msg("no supplier address available for websocket relay")
		return
	}

	// Marshal the original request body for relay processing
	reqBytes, err := req.Marshal()
	if err != nil {
		b.logger.Warn().Err(err).Msg("failed to marshal relay request")
		return
	}

	// Use RelayProcessor if available for proper relay construction
	if b.relayProcessor != nil {
		msg, procErr := b.relayProcessor.ProcessRelay(
			b.ctx,
			reqBytes,
			respPayload,
			supplierAddr,
			b.serviceID,
			b.arrivalHeight,
		)
		if procErr != nil {
			b.logger.Warn().Err(procErr).Msg("failed to process websocket relay")
			return
		}

		if msg != nil {
			if pubErr := b.publisher.Publish(b.ctx, msg); pubErr != nil {
				b.logger.Warn().Err(pubErr).Msg("failed to publish websocket relay")
				return
			}
			wsRelaysEmitted.WithLabelValues(b.serviceID).Inc()
			b.logger.Debug().
				Uint64("relay_count", count).
				Str(logging.FieldSupplier, supplierAddr).
				Msg("websocket relay published")
		}
		return
	}

	// Fallback: Create basic relay message without full processing
	relay := &servicetypes.Relay{
		Req: req,
		Res: resp,
	}
	relayBytes, err := relay.Marshal()
	if err != nil {
		b.logger.Warn().Err(err).Msg("failed to marshal relay")
		return
	}

	msg := &transport.MinedRelayMessage{
		RelayHash:               nil, // Not calculated in fallback mode
		RelayBytes:              relayBytes,
		ComputeUnitsPerRelay:    1,
		SessionId:               "",
		SessionEndHeight:        0,
		SupplierOperatorAddress: supplierAddr,
		ServiceId:               b.serviceID,
		ApplicationAddress:      "",
		ArrivalBlockHeight:      b.arrivalHeight,
	}

	if req.Meta.SessionHeader != nil {
		msg.SessionId = req.Meta.SessionHeader.SessionId
		msg.SessionEndHeight = req.Meta.SessionHeader.SessionEndBlockHeight
		msg.ApplicationAddress = req.Meta.SessionHeader.ApplicationAddress
	}

	msg.SetPublishedAt()

	if pubErr := b.publisher.Publish(b.ctx, msg); pubErr != nil {
		b.logger.Warn().Err(pubErr).Msg("failed to publish websocket relay")
		return
	}

	wsRelaysEmitted.WithLabelValues(b.serviceID).Inc()
	b.logger.Debug().
		Uint64("relay_count", count).
		Str(logging.FieldSupplier, supplierAddr).
		Msg("websocket relay published (fallback)")
}

// tryEmitRelay is deprecated - use emitRelay directly.
// Kept for compatibility but does nothing.
func (b *WebSocketBridge) tryEmitRelay() {
	// No-op: relay emission is now done in handleBackendMessage
}

// setLatestRequest stores the latest request.
func (b *WebSocketBridge) setLatestRequest(req *servicetypes.RelayRequest) {
	b.latestMu.Lock()
	defer b.latestMu.Unlock()
	b.latestRequest = req
}

// getLatestRequest retrieves the latest request.
func (b *WebSocketBridge) getLatestRequest() *servicetypes.RelayRequest {
	b.latestMu.RLock()
	defer b.latestMu.RUnlock()
	return b.latestRequest
}

// setLatestResponse stores the latest response.
func (b *WebSocketBridge) setLatestResponse(resp *servicetypes.RelayResponse) {
	b.latestMu.Lock()
	defer b.latestMu.Unlock()
	b.latestResponse = resp
}

// getLatestResponse retrieves the latest response.
func (b *WebSocketBridge) getLatestResponse() *servicetypes.RelayResponse {
	b.latestMu.RLock()
	defer b.latestMu.RUnlock()
	return b.latestResponse
}

// clearLatestRequest clears the latest request after emitting a relay.
// This ensures each backend response is paired with a unique gateway request.
func (b *WebSocketBridge) clearLatestRequest() {
	b.latestMu.Lock()
	defer b.latestMu.Unlock()
	b.latestRequest = nil
}

// Close shuts down the WebSocket bridge.
func (b *WebSocketBridge) Close() error {
	if !b.closed.CompareAndSwap(false, true) {
		return nil // Already closed
	}

	// Decrement active connections metric
	wsConnectionsActive.WithLabelValues(b.serviceID).Dec()

	// Log final stats for this connection
	relayCount := b.relayCount.Load()
	if relayCount > 0 {
		b.logger.Info().
			Uint64("relays_emitted", relayCount).
			Msg("websocket bridge closing with relays emitted")
	}

	b.cancelFn()

	// Send close messages (best-effort, errors are logged but not propagated)
	closeMsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bridge closing")
	deadline := time.Now().Add(wsWriteWait)

	if err := b.gatewayConn.WriteControl(websocket.CloseMessage, closeMsg, deadline); err != nil {
		b.logger.Debug().Err(err).Msg("failed to send close to gateway")
	}
	if err := b.backendConn.WriteControl(websocket.CloseMessage, closeMsg, deadline); err != nil {
		b.logger.Debug().Err(err).Msg("failed to send close to backend")
	}

	// Give connections time to receive close message
	time.Sleep(100 * time.Millisecond)

	b.gatewayConn.Close()
	b.backendConn.Close()

	b.logger.Debug().Msg("websocket bridge closed")
	return nil
}

// WebSocketHandler returns an HTTP handler for WebSocket upgrades.
// This should be used when detecting WebSocket upgrade requests.
func (p *ProxyServer) WebSocketHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serviceID := p.extractServiceID(r)
		if serviceID == "" {
			p.sendError(w, http.StatusBadRequest, "missing service ID")
			return
		}

		svcConfig, ok := p.config.Services[serviceID]
		if !ok {
			p.sendError(w, http.StatusNotFound, "unknown service")
			return
		}

		// Upgrade HTTP connection to WebSocket
		gatewayConn, err := WebSocketUpgrader.Upgrade(w, r, nil)
		if err != nil {
			p.logger.Warn().Err(err).Msg("failed to upgrade to websocket")
			return
		}

		// Get WebSocket backend URL
		var backendURL string
		var configHeaders map[string]string
		if backend, ok := svcConfig.Backends["websocket"]; ok {
			backendURL = backend.URL
			configHeaders = backend.Headers
		} else {
			http.Error(w, "WebSocket backend not configured for this service", http.StatusServiceUnavailable)
			return
		}

		// Build headers
		headers := make(http.Header)
		for k, v := range configHeaders {
			headers.Set(k, v)
		}

		arrivalHeight := p.currentBlockHeight.Load()

		// Create and run bridge
		bridge, err := NewWebSocketBridge(
			p.logger,
			gatewayConn,
			backendURL,
			serviceID,
			p.supplierAddress,
			arrivalHeight,
			p.relayProcessor,
			p.publisher,
			p.responseSigner,
			headers,
		)
		if err != nil {
			p.logger.Warn().Err(err).Msg("failed to create websocket bridge")
			gatewayConn.Close()
			return
		}

		// Run bridge (blocking)
		bridge.Run()
	}
}

// IsWebSocketUpgrade checks if the request is a WebSocket upgrade request.
func IsWebSocketUpgrade(r *http.Request) bool {
	return websocket.IsWebSocketUpgrade(r)
}
