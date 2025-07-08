package proxy

import (
	"context"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/relayer/config"
	proxyws "github.com/pokt-network/poktroll/pkg/relayer/proxy/websockets"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// handleAsyncConnection handles the asynchronous relay request by creating a
// websocket bridge between the client and the service endpoint.
func (server *relayMinerHTTPServer) handleAsyncConnection(
	ctx context.Context,
	writer http.ResponseWriter,
	request *http.Request,
) error {
	// Determine the service ID and application address from the request headers.
	serviceId := request.Header.Get("Target-Service-Id")
	appAddress := request.Header.Get("App-Address")

	// Determine the supplier's service configuration.
	supplierConfig, ok := server.serverConfig.SupplierConfigsMap[serviceId]
	if !ok {
		return ErrRelayerProxyServiceEndpointNotHandled
	}

	// Get the websocket service config. We can safely use the `config.RPCTypeWS`
	// as `handleAsyncConnection` is only called for requests with the 'Rpc-Type'
	// header set to 'websocket'.
	//
	// IMPORTANT: This will return an error if the service is not configured for websocket RPC type.
	websocketServiceConfig, ok := supplierConfig.RPCTypeServiceConfigs[config.RPCTypeWS]
	if !ok {
		return ErrRelayerProxyServiceEndpointNotHandled.Wrapf(
			"service %q not configured for websocket RPC type",
			serviceId,
		)
	}

	logger := server.logger.With(
		"relay_request_type", "asynchronous",
		"service_id", serviceId,
		"application_address", appAddress,
	)

	// Get the current height session to determine the session parameters.
	block := server.blockClient.LastBlock(ctx)

	logger.ProbabilisticDebugInfo(polylog.ProbabilisticDebugInfoProb).Msgf(
		"📊 Chain head at height %d (block hash: %X) during WebSocket session setup",
		block.Height(),
		block.Hash(),
	)

	session, err := server.sessionQueryClient.GetSession(ctx, appAddress, serviceId, block.Height())
	if err != nil {
		return ErrRelayerProxyInternalError.Wrapf("error getting session: %v", err)
	}

	sessionHeader := session.Header

	logger = logger.With(
		"server_addr", server.server.Addr,
		"session_start_height", sessionHeader.SessionStartBlockHeight,
		"destination_url", websocketServiceConfig.BackendUrl.String(),
	)

	// Upgrade the HTTP connection to a websocket connection.
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	clientConn, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		logger.Error().Err(err).Msg("upgrading connection to websocket")
		return ErrRelayerProxyInternalError.Wrap(err.Error())
	}

	// TODO_MAINNET(@red0ne): Add unit and e2e tests for the websocket bridge and connection.
	// Create a new websocket bridge between the gateway and the service endpoint.
	bridge, err := proxyws.NewBridge(
		logger,
		server.relayAuthenticator,
		server.relayMeter,
		server.servedRewardableRelaysProducer,
		server.blockClient,
		websocketServiceConfig,
		session,
		clientConn,
	)
	if err != nil {
		logger.Error().Err(err).Msg("creating websocket bridge")
		return ErrRelayerProxyInternalError.Wrap(err.Error())
	}

	// Set up the bridge to close before the claim window opens.
	// TODO_CONSIDERATION: Async connection could be stricter and close the bridge
	// right after the session ends, but it is technically possible to delay it
	// until the claim window opening height to maximize profit for the supplier
	// and delay reconnecting the upstream client as much as possible.
	sharedParams, err := server.sharedQueryClient.GetParams(ctx)
	if err != nil {
		return ErrRelayerProxyInternalError.Wrap(err.Error())
	}
	sessionEndHeight := sessionHeader.SessionEndBlockHeight
	claimWindowOpenHeight := sharedtypes.GetClaimWindowOpenHeight(sharedParams, sessionEndHeight)

	// Run the websockets bridge.
	// Set up the bridge to close after the session ends.
	go bridge.Run(claimWindowOpenHeight)

	logger.Info().Msg("websocket connection established with client")

	return nil
}
