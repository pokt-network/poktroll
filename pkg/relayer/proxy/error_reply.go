package proxy

import (
	"net/http"

	"github.com/pokt-network/shannon-sdk/httpcodec"
	"github.com/pokt-network/shannon-sdk/rpcdetect"

	"github.com/pokt-network/poktroll/x/service/types"
)

// replyWithError builds the appropriate error format according to the payload
// using the passed in error and writes it to the writer.
// NOTE: This method is used to reply with an "internal" error that is related
// to the server itself and not to the relayed request.
func (sync *synchronousRPCServer) replyWithError(
	upstreamError error,
	relayRequest *types.RelayRequest,
	isInternalError bool,
	writer http.ResponseWriter,
) {
	serviceId := relayRequest.Meta.SessionHeader.Service.Id

	httpRequest, _ := httpcodec.DeserializeHTTPRequest(relayRequest.Payload)

	rpcType := rpcdetect.GetRPCType(httpRequest)
	_, errorResponseBz := rpcdetect.FormatError(upstreamError, httpRequest, rpcType, isInternalError)

	relaysErrorsTotal.With("service_id", serviceId, "rpc_type", rpcType.String()).Add(1)

	relayResponse := &types.RelayResponse{Payload: errorResponseBz}
	relayResponseBz, err := relayResponse.Marshal()
	if err != nil {
		sync.replyWithDefaultError(err, serviceId, writer)
		return
	}

	if _, upstreamError = writer.Write(relayResponseBz); upstreamError != nil {
		sync.replyWithDefaultError(upstreamError, serviceId, writer)
		return
	}
}

func (sync *synchronousRPCServer) replyWithDefaultError(
	err error,
	serviceId string,
	writer http.ResponseWriter,
) {
	listenAddress := sync.serverConfig.ListenAddress

	sync.logger.Error().Err(err).
		Str("service_id", serviceId).
		Str("listen_address", listenAddress).
		Msg("failed generating error response")

	writer.WriteHeader(http.StatusInternalServerError)

	if _, err = writer.Write([]byte("internal server error")); err != nil {
		sync.logger.Error().Err(err).
			Str("service_id", serviceId).
			Str("listen_address", listenAddress).
			Msg("failed writing default error response")
	}
}
