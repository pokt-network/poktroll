package appgateserver

import (
	"context"
	"net/http"

	"github.com/pokt-network/poktroll/pkg/partials"
)

// replyWithError replies to the application with an error response and writes
// it to the writer provided.
// NOTE: This method is used to reply with an "internal" error that is related
// to the appgateserver itself and not to the relay request.
func (app *appGateServer) replyWithError(
	ctx context.Context,
	payloadBz []byte,
	writer http.ResponseWriter,
	serviceId string,
	rpcType string,
	err error,
) {
	relaysErrorsTotal.With("service_id", serviceId, "rpc_type", rpcType).Add(1)
	responseBz, err := partials.GetErrorReply(ctx, payloadBz, err)
	if err != nil {
		app.logger.Error().Err(err).Str("service_id", serviceId).Msg("failed getting error reply")
		return
	}

	if _, err = writer.Write(responseBz); err != nil {
		app.logger.Error().Err(err).Str("service_id", serviceId).Msg("failed writing relay response")
		return
	}
}
