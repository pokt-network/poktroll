package websockets

import (
	"context"
	"sync"

	"github.com/gorilla/websocket"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/config"
	"github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

// bridge represents a websocket bridge between the gateway and the service backend.
// It is responsible for forwarding relay requests from the gateway to the service
// backend and relay responses from the service backend to the gateway.
//
// Due to the asynchronous nature of websockets, there isn't always a 1:1 mapping
// between requests and responses. The bridge must handle two scenarios:
//
//  1. Many responses for few requests (M-resp >> N-req):
//     Example: A client subscribes once to an event stream (eth_subscribe) but
//     receives many event notifications over time.
//
//  2. Many requests for few responses (N-req >> M-resp):
//     Example: A client uploads a large file in chunks, sending many requests,
//     while the server only occasionally sends progress updates.
//
// This design has two important implications:
//
//  1. Each message (inbound or outbound) is treated as a reward-eligible relay.
//     For example, with eth_subscribe, both the initial subscription request and
//     each received event would be eligible for rewards.
//
//  2. To maintain protocol compatibility, the bridge must always pair messages
//     when submitting to the miner. It does this by combining the most recent
//     request with the most recent response.
//
// TODO_FUTURE: Currently, the RelayMiner is paid for each incoming and outgoing
// message transmitted.
// While this is the most common and trivial use case, future services might have
// different payable units of work (e.g. packet size, specific packet or data delimiter...).
// To support these use cases, the bridge should be extensible to allow for custom
// units of work to be metered and paid.
type bridge struct {
	ctx       context.Context
	cancelCtx context.CancelFunc
	logger    polylog.Logger

	// serviceBackendConn is the websocket connection to the service backend.
	serviceBackendConn *connection

	// gatewayConn is the websocket connection to the gateway.
	gatewayConn *connection

	// msgChan is the channel that the bridge uses to communicate with the connections.
	msgChan <-chan message

	// relayAuthenticator is the relay authenticator that the bridge uses to verify
	// relay requests and sign relay responses.
	relayAuthenticator relayer.RelayAuthenticator

	// relayMeter is the relay meter that the bridge uses to estimate the relay reward
	// of the bridged relay requests.
	relayMeter relayer.RelayMeter

	// blockClient is the client used to get the latest block height.
	blockClient client.BlockClient

	// latestRelayRequest is the latest relay request received from the gateway.
	// It is used to emit relays to the miner such that there is always a request/response
	// pair available when submitting proofs.
	// This is particularly important for asynchronous communication where it is
	// mostly or exclusively the service backend that sends messages to the gateway
	// such as eth_subscribe.
	latestRelayRequest *types.RelayRequest

	// latestRelayResponse is the latest relay response received from the service backend.
	// It is used to emit relays to the miner such that there is always a request/response
	// pair available when submitting proofs.
	// This is particularly important for asynchronous communication where it is
	// mostly or exclusively the gateway that sends messages to the service backend
	// such as file upload protocols.
	latestRelayResponse *types.RelayResponse

	// latestRelayMu is the mutex that protects the latest relay request and response.
	latestRelayMu sync.RWMutex

	// relaysProducer is the channel that the bridge uses to emit the relays that
	// have been served to the miner.
	relaysProducer chan<- *types.Relay

	// session is the session that the bridge is serving.
	// It ensures that the bridge only serves relay requests matching the session
	// it was created for.
	session *sessiontypes.Session
}

// NewBridge creates a new websocket bridge between the gateway and the service backend.
func NewBridge(
	logger polylog.Logger,
	relayAuthenticator relayer.RelayAuthenticator,
	relayMeter relayer.RelayMeter,
	serverRelaysProducer chan<- *types.Relay,
	blockClient client.BlockClient,
	serviceConfig *config.RelayMinerSupplierServiceConfig,
	session *sessiontypes.Session,
	gatewayWSConn *websocket.Conn,
) (*bridge, error) {
	bridgeLogger := logger.With("component", "bridge")

	// Connect to the service backend.
	serviceBackendWSConn, err := ConnectServiceBackend(serviceConfig.BackendUrl, serviceConfig.HeadersHTTP())
	if err != nil {
		bridgeLogger.Error().Err(err).Msg("failed to connect to the service backend")
		return nil, ErrWebsocketsBridge.Wrapf("failed to connect to the service backend: %v", err)
	}

	// Using an observable prevents race conditions during connection termination:
	//  - Both serviceBackendConn and gatewayConn need to read the stop signal
	//  - Direct stopChan reads could cause one connection to miss the signal during simultaneous reads
	//  - Observable pattern ensures all components receive termination notifications
	stopBridgeObservable, stopChan := channel.NewObservable[error]()
	msgChan := make(chan message)
	ctx, cancelCtx := context.WithCancel(context.Background())

	// Create the service backend connection manager.
	serviceBackendConn := newConnection(
		ctx,
		serviceBackendWSConn,
		bridgeLogger,
		messageSourceServiceBackend,
		session.Header.ServiceId,
		msgChan,
		stopChan,
		stopBridgeObservable,
	)

	// Create the gateway connection manager.
	gatewayConn := newConnection(
		ctx,
		gatewayWSConn,
		bridgeLogger,
		messageSourceGateway,
		session.Header.ServiceId,
		msgChan,
		stopChan,
		stopBridgeObservable,
	)

	bridge := &bridge{
		ctx:                ctx,
		cancelCtx:          cancelCtx,
		logger:             bridgeLogger,
		serviceBackendConn: serviceBackendConn,
		gatewayConn:        gatewayConn,
		msgChan:            msgChan,
		relayAuthenticator: relayAuthenticator,
		relayMeter:         relayMeter,
		relaysProducer:     serverRelaysProducer,
		blockClient:        blockClient,
		session:            session,
	}

	return bridge, nil
}

// Run initiates the message loop of the bridge.
// It is a blocking method and should be called in a goroutine.
// It is scheduled to stop when the closeHeight is reached.
func (b *bridge) Run(closeHeight int64) {
	go b.messageLoop()

	channel.ForEach(
		b.ctx,
		b.blockClient.CommittedBlocksSequence(b.ctx),
		func(ctx context.Context, block client.Block) {
			if block.Height() >= closeHeight {
				b.logger.Info().Msg("session closing, bridge stopped")
				// TODO_MAINNET(@red-0ne) Propagate the session closing as a close message
				// to the gateway so end user can be informed.
				b.cancelCtx()
			}
		},
	)

	b.logger.Info().Msg("bridge started")
}

// messageLoop is the main loop of the bridge:
//   - It listens for messages from the gateway and forwards them to the service backend,
//   - It listens for messages from the service backend and forwards them to the gateway.
func (b *bridge) messageLoop() {
	for {
		select {
		case <-b.ctx.Done():
			return
		case msg := <-b.msgChan:
			switch msg.source {

			// message from gateway sent to service over websocket
			case messageSourceGateway:
				b.handleGatewayIncomingMessage(msg)

			// message from service sent to gateway over websocket
			case messageSourceServiceBackend:
				b.handleServiceBackendIncomingMessage(msg)
			}
		}
	}
}

// handleGatewayIncomingMessage handles incoming messages from the gateway.
// It receives relay requests from the gateway, verifies them, forwards their payloads
// to the service backend.
func (b *bridge) handleGatewayIncomingMessage(msg message) {
	logger := b.logger.With(
		"message_source", messageSourceGateway,
		"message_type", msg.messageType,
	)

	logger.Debug().Msg("received message from gateway")

	// Unmarshal msg.data into a RelayRequest.
	var relayRequest types.RelayRequest
	if err := relayRequest.Unmarshal(msg.data); err != nil {
		b.serviceBackendConn.handleError(
			ErrWebsocketsGatewayMessage.Wrapf("failed to unmarshal relay request: %v", err),
		)
		return
	}

	// Ensure that the relay request is for the session that the bridge is serving.
	if relayRequest.Meta.SessionHeader.SessionId != b.session.Header.SessionId {
		b.serviceBackendConn.handleError(
			ErrWebsocketsGatewayMessage.Wrapf(
				"the relay request session id %q does not match the bridge session id %q",
				relayRequest.Meta.SessionHeader.SessionId, b.session.Header.SessionId,
			),
		)
		return
	}

	serviceId := relayRequest.Meta.SessionHeader.ServiceId

	// Store the latest relay request to guarantee that there is always a request/response.
	// This is to guarantee that any mined Relay will contain a request/response pair
	// which is a requirement for the protocol's proof verification.
	// E.g. The latest eth_subscribe RelayRequest will be mapped to multiple responses.
	b.setLatestRelayRequest(&relayRequest)

	relayer.RelaysTotal.With(
		"service_id", serviceId,
		"supplier_operator_address", relayRequest.Meta.SupplierOperatorAddress,
	).Add(1)

	relayer.RelayRequestSizeBytes.With("service_id", serviceId).
		Observe(float64(relayRequest.Size()))

	// Verify the relay request signature and session.
	if err := b.relayAuthenticator.VerifyRelayRequest(b.ctx, &relayRequest, serviceId); err != nil {
		b.serviceBackendConn.handleError(
			ErrWebsocketsGatewayMessage.Wrapf("failed to verify relay request: %v", err),
		)
		return
	}

	logger.Debug().Msg("relay request verified")

	// Forward the relay request payload to the service backend.
	if err := b.serviceBackendConn.WriteMessage(msg.messageType, relayRequest.Payload); err != nil {
		b.serviceBackendConn.handleError(
			ErrWebsocketsGatewayMessage.Wrapf("failed to send relay request to service backend: %v", err),
		)
		return
	}

	logger.Debug().Msg("relay request forwarded to service backend")

	// Do not emit a relay to the miner if there is no response to form a request/response pair.
	if b.getLatestRelayResponse() == nil {
		logger.Info().Msg("waiting for service backend response")
		return
	}

	relay := &types.Relay{
		Req: &relayRequest,
		Res: b.getLatestRelayResponse(),
	}

	relayer.RelaysSuccessTotal.With("service_id", serviceId).Add(1)

	// Emit the relay to the miner.
	// Since async relays might be request or response only, each request or response
	// is considered to be eligible for a reward.
	b.relaysProducer <- relay

	logger.Debug().Msg("relay emitted to miner")

	// Accumulate the relay reward.
	// The asynchronous flow assumes that every inbound and outbound message is a
	// payment-eligible relay.
	// Recall that num inbound messages is unlikely to equal num outbound messages in a websocket.
	if err := b.relayMeter.AccumulateRelayReward(b.ctx, relayRequest.Meta); err != nil {
		b.serviceBackendConn.handleError(
			ErrWebsocketsGatewayMessage.Wrapf("failed to accumulate relay reward: %v", err),
		)
		return
	}

}

// handleServiceBackendIncomingMessage handles incoming messages from the service backend.
// It receives relay responses from the service backend, signs them, and forwards them
// to the gateway.
func (b *bridge) handleServiceBackendIncomingMessage(msg message) {
	logger := b.logger.With(
		"message_source", messageSourceServiceBackend,
		"message_type", msg.messageType,
	)

	logger.Debug().Msg("received message from service backend")

	// Use the latest relay request's session header to create the RelayResponse.
	meta := b.latestRelayRequest.Meta
	serviceId := meta.SessionHeader.ServiceId

	// Create a RelayResponse from the service backend message.
	relayResponse := &types.RelayResponse{
		Meta:    types.RelayResponseMetadata{SessionHeader: meta.SessionHeader},
		Payload: msg.data,
	}

	relayer.RelaysTotal.With(
		"service_id", serviceId,
		"supplier_operator_address", meta.SupplierOperatorAddress,
	).Add(1)

	relayer.RelayResponseSizeBytes.With("service_id", serviceId).
		Observe(float64(relayResponse.Size()))

	// Sign the relay response and add the signature to the relay response metadata
	if err := b.relayAuthenticator.SignRelayResponse(relayResponse, meta.SupplierOperatorAddress); err != nil {
		b.gatewayConn.handleError(
			ErrWebsocketsServiceBackendMessage.Wrapf("failed to sign relay response: %v", err),
		)
		return
	}

	logger.Debug().Msg("relay response signed")

	// Store the latest relay response to guarantee that there is always a request/response.
	// This is to guarantee that any mined Relay will contain a request/response pair
	// which is a requirement for the protocol's proof verification.
	// E.g. The latest RelayResponse will be mapped to the initial eth_subscribe RelayRequest.
	b.setLatestRelayResponse(relayResponse)

	relayResponseBz, err := relayResponse.Marshal()
	if err != nil {
		b.gatewayConn.handleError(
			ErrWebsocketsServiceBackendMessage.Wrapf("failed to marshal relay response: %v", err),
		)
		return
	}

	// Forward the relay response to the gateway.
	if err := b.gatewayConn.WriteMessage(msg.messageType, relayResponseBz); err != nil {
		b.gatewayConn.handleError(
			ErrWebsocketsServiceBackendMessage.Wrapf("failed to send relay response to gateway: %v", err),
		)
		return
	}

	logger.Debug().Msg("relay response forwarded to gateway")

	// Do not emit a relay to the miner if there is no response to form a request/response pair.
	if b.getLatestRelayRequest() == nil {
		logger.Info().Msg("waiting for service backend request")
		return
	}

	relay := &types.Relay{
		Req: b.getLatestRelayRequest(),
		Res: relayResponse,
	}

	relayer.RelaysSuccessTotal.With("service_id", serviceId).Add(1)

	// Emit the relay to the miner.
	// Since async relays might be request or response only, each request or response
	// is considered to be eligible for a reward.
	b.relaysProducer <- relay

	logger.Debug().Msg("relay emitted to miner")

	// Accumulate the relay reward.
	// The asynchronous flow assumes that every inbound and outbound message is a
	// payment-eligible relay.
	// Recall that num inbound messages is unlikely to equal num outbound messages in a websocket.
	if err := b.relayMeter.AccumulateRelayReward(b.ctx, b.latestRelayRequest.Meta); err != nil {
		b.gatewayConn.handleError(
			ErrWebsocketsServiceBackendMessage.Wrapf("failed to accumulate relay reward: %v", err),
		)
		return
	}
}

// setLatestRelayRequest sets the latest relay request in a concurrency-safe manner.
func (b *bridge) setLatestRelayRequest(relayRequest *types.RelayRequest) {
	b.latestRelayMu.Lock()
	defer b.latestRelayMu.Unlock()

	b.latestRelayRequest = relayRequest
}

// getLatestRelayRequest gets the latest relay request in a concurrency-safe manner.
func (b *bridge) getLatestRelayRequest() *types.RelayRequest {
	b.latestRelayMu.RLock()
	defer b.latestRelayMu.RUnlock()

	return b.latestRelayRequest
}

// setLatestRelayResponse sets the latest relay response in a concurrency-safe manner.
func (b *bridge) setLatestRelayResponse(relayResponse *types.RelayResponse) {
	b.latestRelayMu.Lock()
	defer b.latestRelayMu.Unlock()

	b.latestRelayResponse = relayResponse
}

// getLatestRelayResponse gets the latest relay response in a concurrency-safe manner.
func (b *bridge) getLatestRelayResponse() *types.RelayResponse {
	b.latestRelayMu.RLock()
	defer b.latestRelayMu.RUnlock()

	return b.latestRelayResponse
}
