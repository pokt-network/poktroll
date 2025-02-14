package websockets

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/url"
	"path"

	"github.com/gorilla/websocket"
	sdktypes "github.com/pokt-network/shannon-sdk/types"

	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/config"
	"github.com/pokt-network/poktroll/x/service/types"
)

type bridge struct {
	logger             polylog.Logger
	serviceBackendConn *connection
	gatewayConn        *connection
	msgChan            chan message
	stopChan           chan error

	relayAuthenticator   relayer.RelayAuthenticator
	relayMeter           relayer.RelayMeter
	latestRelayRequest   *types.RelayRequest
	latestRelayResponse  *types.RelayResponse
	servedRelaysProducer chan<- *types.Relay
}

func NewBridge(
	ctx context.Context,
	logger polylog.Logger,
	relayAuthenticator relayer.RelayAuthenticator,
	relayMeter relayer.RelayMeter,
	serverRelaysProducer chan<- *types.Relay,
	serviceConfig *config.RelayMinerSupplierServiceConfig,
	relayRequest *types.RelayRequest,
	gatewayWSConn *websocket.Conn,
) (*bridge, error) {
	sessionHeader := relayRequest.Meta.SessionHeader
	if err := relayAuthenticator.VerifyRelayRequest(ctx, relayRequest, sessionHeader.ServiceId); err != nil {
		return nil, err
	}

	// Deserialize the relay request payload to get the upstream HTTP request.
	poktHTTPRequest, err := sdktypes.DeserializeHTTPRequest(relayRequest.Payload)
	if err != nil {
		return nil, err
	}

	serviceBackendUrl, headers, err := buildServiceBackendRequest(poktHTTPRequest, serviceConfig)
	if err != nil {
		return nil, err
	}

	serviceBackendWSConn, err := connectServiceBackend(serviceBackendUrl, headers)
	if err != nil {
		return nil, err
	}

	msgChan := make(chan message)
	stopChan := make(chan error)

	bridgeLogger := logger.With("")

	serviceBackendConn := newConnection(
		logger.With("connection", "service_backend"),
		serviceBackendWSConn,
		messageSourceServiceBackend,
		msgChan,
		stopChan,
	)

	gatewayConn := newConnection(
		logger.With("connection", "gateway"),
		gatewayWSConn,
		messageSourceGateway,
		msgChan,
		stopChan,
	)

	bridge := &bridge{
		logger:               bridgeLogger,
		serviceBackendConn:   serviceBackendConn,
		gatewayConn:          gatewayConn,
		msgChan:              make(chan message),
		stopChan:             make(chan error),
		latestRelayRequest:   relayRequest,
		relayAuthenticator:   relayAuthenticator,
		relayMeter:           relayMeter,
		servedRelaysProducer: serverRelaysProducer,
	}

	return bridge, nil
}

func (b *bridge) Run(ctx context.Context) {
	go b.messageLoop(ctx)

	b.logger.Info().Msg("bridge started")

	<-b.stopChan
}

func (b *bridge) Close() {
	close(b.stopChan)
}

func (b *bridge) messageLoop(ctx context.Context) {
	for {
		select {
		case <-b.stopChan:
			return

		case msg := <-b.msgChan:
			switch msg.source {

			case messageSourceGateway:
				b.handleGatewayMessage(ctx, msg)

			case messageSourceServiceBackend:
				b.handleServiceBackendMessage(ctx, msg)
			}
		}
	}
}

func (b *bridge) handleGatewayMessage(ctx context.Context, msg message) {
	b.logger.Debug().Msg("received message from gateway")

	var relayRequest types.RelayRequest
	if err := relayRequest.Unmarshal(msg.data); err != nil {
		b.serviceBackendConn.handleError(err, messageSourceGateway)
		return
	}

	b.latestRelayRequest = &relayRequest

	serviceId := relayRequest.Meta.SessionHeader.ServiceId

	if err := b.relayAuthenticator.VerifyRelayRequest(ctx, &relayRequest, serviceId); err != nil {
		b.serviceBackendConn.handleError(err, messageSourceGateway)
		return
	}

	if err := b.serviceBackendConn.WriteMessage(msg.messageType, relayRequest.Payload); err != nil {
		b.serviceBackendConn.handleError(err, messageSourceServiceBackend)
		return
	}

	if b.latestRelayResponse == nil {
		b.logger.Debug().Msg("waiting for service backend response")
		return
	}

	relay := &types.Relay{
		Req: &relayRequest,
		Res: b.latestRelayResponse,
	}

	b.servedRelaysProducer <- relay

	if err := b.relayMeter.AccumulateRelayReward(ctx, relayRequest.Meta); err != nil {
		b.serviceBackendConn.handleError(err, messageSourceGateway)
		return
	}

}

func (b *bridge) handleServiceBackendMessage(ctx context.Context, msg message) {
	b.logger.Debug().Msg("received message from service backend")

	meta := b.latestRelayRequest.Meta

	relayResponse := &types.RelayResponse{
		Meta:    types.RelayResponseMetadata{SessionHeader: meta.SessionHeader},
		Payload: msg.data,
	}

	// Sign the relay response and add the signature to the relay response metadata
	if err := b.relayAuthenticator.SignRelayResponse(relayResponse, meta.SupplierOperatorAddress); err != nil {
		b.gatewayConn.handleError(err, messageSourceGateway)
		return
	}

	b.latestRelayResponse = relayResponse

	relayResponseBz, err := relayResponse.Marshal()
	if err != nil {
		b.gatewayConn.handleError(err, messageSourceGateway)
		return
	}

	if err := b.gatewayConn.WriteMessage(msg.messageType, relayResponseBz); err != nil {
		b.gatewayConn.handleError(err, messageSourceGateway)
	}

	relay := &types.Relay{
		Req: b.latestRelayRequest,
		Res: relayResponse,
	}

	b.servedRelaysProducer <- relay

	if err := b.relayMeter.AccumulateRelayReward(ctx, b.latestRelayRequest.Meta); err != nil {
		b.serviceBackendConn.handleError(err, messageSourceGateway)
		return
	}
}

func buildServiceBackendRequest(
	poktHTTPRequest *sdktypes.POKTHTTPRequest,
	serviceConfig *config.RelayMinerSupplierServiceConfig,
) (*url.URL, http.Header, error) {
	serviceBackendUrl, err := url.Parse(poktHTTPRequest.Url)
	if err != nil {
		return nil, nil, err
	}

	serviceBackendUrl.Host = serviceConfig.BackendUrl.Host
	serviceBackendUrl.Scheme = serviceConfig.BackendUrl.Scheme

	// Prepend the path of the service's backend URL to the path of the upstream request.
	// This is done to ensure that the request complies with the service's backend URL,
	// while preserving the path of the original request.
	// This is particularly important for RESTful APIs where the path is used to
	// determine the resource being accessed.
	// For example, if the service's backend URL is "http://host:8080/api/v1",
	// and the upstream request path is "/users", the final request path will be
	// "http://host:8080/api/v1/users".
	serviceBackendUrl.Path = path.Join(serviceConfig.BackendUrl.Path, serviceBackendUrl.Path)

	// Merge the query parameters of the upstream request with the query parameters
	// of the service's backend URL.
	// This is done to ensure that the query parameters of the original request are
	// passed and that the service's backend URL query parameters are also included.
	// This is important for RESTful APIs where query parameters are used to filter
	// and paginate resources.
	// For example, if the service's backend URL is "http://host:8080/api/v1?key=abc",
	// and the upstream request has a query parameter "page=1", the final request URL
	// will be "http://host:8080/api/v1?key=abc&page=1".
	query := serviceBackendUrl.Query()
	for key, values := range serviceConfig.BackendUrl.Query() {
		for _, value := range values {
			query.Add(key, value)
		}
	}
	serviceBackendUrl.RawQuery = query.Encode()

	// Create the HTTP header for the request by converting the RelayRequest's
	// POKTHTTPRequest.Header to an http.Header.
	header := http.Header{}
	poktHTTPRequest.CopyToHTTPHeader(header)

	if serviceConfig.Authentication != nil {
		auth := serviceConfig.Authentication.Username + ":" + serviceConfig.Authentication.Password
		header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(auth)))
	}

	// Add any service configuration specific headers to the request, such as
	// authentication or authorization headers. These will override any upstream
	// request headers with the same key.
	for key, value := range serviceConfig.Headers {
		header.Set(key, value)
	}

	return serviceBackendUrl, header, nil
}
