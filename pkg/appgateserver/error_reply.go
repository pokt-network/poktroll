package appgateserver

import (
	"net/http"
	"strings"

	"github.com/pokt-network/shannon-sdk/httpcodec"
	"github.com/pokt-network/shannon-sdk/rpcdetect"
)

// replyWithError replies to the application with an error response and writes
// it to the writer provided.
// NOTE: This method is used to reply with an "internal" error that is related
// to the appgateserver itself and not to the relay request.
func (app *appGateServer) replyWithError(
	upstreamError error,
	httpRequest *httpcodec.HTTPRequest,
	isInternalError bool,
	writer http.ResponseWriter,
	serviceId string,
) {
	rpcType := rpcdetect.GetRPCType(httpRequest)
	errorResponse, _ := rpcdetect.FormatError(upstreamError, httpRequest, rpcType, isInternalError)

	relaysErrorsTotal.With("service_id", serviceId, "rpc_type", rpcType.String()).Add(1)

	writer.WriteHeader(int(errorResponse.StatusCode))
	for key, valuesStr := range errorResponse.Header {
		values := strings.Split(valuesStr, ",")
		for _, v := range values {
			writer.Header().Add(key, v)
		}
	}

	if _, upstreamError = writer.Write(errorResponse.Body); upstreamError != nil {
		app.replyWithDefaultError(upstreamError, serviceId, writer)
		return
	}
}

func (app *appGateServer) replyWithDefaultError(
	err error,
	serviceId string,
	writer http.ResponseWriter,
) {
	app.logger.Error().
		Err(err).
		Str("service_id", serviceId).
		Msg("failed generating error response")

	writer.WriteHeader(http.StatusInternalServerError)

	if _, err = writer.Write([]byte("internal server error")); err != nil {
		app.logger.Error().
			Err(err).
			Str("service_id", serviceId).
			Msg("failed writing error response")
	}
}
