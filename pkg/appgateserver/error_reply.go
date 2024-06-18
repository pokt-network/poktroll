package appgateserver

import (
	"net/http"

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

	errorResponse, _ := poktHTTPRequest.FormatError(replyError, false)

	// Write all the errorResponse headers to the writer.
	// While the only header that errorResponse.Header contains is "Content-Type",
	// errorResponse.Header are iterated over to future-proof against any additional
	// headers that may be added in the future (e.g. compression or caching headers).
	errorResponse.CopyToHTTPHeader(writer.Header())
	writer.WriteHeader(int(errorResponse.StatusCode))

	if _, err := writer.Write(errorResponse.BodyBz); err != nil {
		app.logger.Error().Err(err).Str("service_id", serviceId).Msg("failed writing relay response")
		return
	}
}
