package appgateserver

import (
	"net/http"

	"github.com/pokt-network/shannon-sdk/rpcdetect"
	sdktypes "github.com/pokt-network/shannon-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// replyWithError replies to the application with an error response and writes
// it to the writer provided.
// NOTE: This method is used to reply with an "internal" error that is related
// to the appgateserver itself and not to the relay request.
func (app *appGateServer) replyWithError(
	replyError error,
	poktHTTPRequest *sdktypes.POKTHTTPRequest,
	serviceId string,
	rpcType sharedtypes.RPCType,
	writer http.ResponseWriter,
) {
	relaysErrorsTotal.With("service_id", serviceId, "rpc_type", rpcType.String()).Add(1)

	errorResponse, _ := rpcdetect.FormatError(replyError, poktHTTPRequest, rpcType, false)

	writer.WriteHeader(int(errorResponse.StatusCode))

	for key, header := range errorResponse.Header {
		for _, value := range header.Values {
			writer.Header().Add(key, value)
		}
	}

	if _, err := writer.Write(errorResponse.BodyBz); err != nil {
		app.logger.Error().Err(err).Str("service_id", serviceId).Msg("failed writing relay response")
		return
	}
}
