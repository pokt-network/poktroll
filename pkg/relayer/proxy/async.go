package proxy

import (
	"context"
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/config"
	proxyws "github.com/pokt-network/poktroll/pkg/relayer/proxy/websockets"
	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// serveHTTP holds the underlying logic of ServeHTTP.
func (server *httpServer) handleAsyncConnection(
	ctx context.Context,
	relayAuthenticator relayer.RelayAuthenticator,
	serviceConfig *config.RelayMinerSupplierServiceConfig,
	sharedParams *sharedtypes.Params,
	relayRequest *types.RelayRequest,
	writer http.ResponseWriter,
	request *http.Request,
) error {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	clientConn, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		server.logger.Error().Err(err).Msg("upgrading connection to websocket")
		return ErrRelayerProxyInternalError.Wrap(err.Error())
	}

	bridge, err := proxyws.NewBridge(
		ctx,
		server.logger,
		relayAuthenticator,
		server.relayMeter,
		server.servedRelaysProducer,
		serviceConfig,
		relayRequest,
		clientConn,
	)

	sessionEndHeight := relayRequest.Meta.SessionHeader.SessionEndBlockHeight
	claimWindowOpenHeight := sharedtypes.GetClaimWindowOpenHeight(sharedParams, sessionEndHeight)
	channel.ForEach(
		ctx,
		server.blockClient.CommittedBlocksSequence(ctx),
		func(ctx context.Context, block client.Block) {
			if block.Height() >= claimWindowOpenHeight {
				bridge.Close()
			}
		},
	)

	if err != nil {
		server.logger.Error().Err(err).Msg("creating websocket bridge")
		return ErrRelayerProxyInternalError.Wrap(err.Error())
	}

	go bridge.Run(ctx)

	server.logger.Info().Msg("websocket connection established with client")

	return nil
}
