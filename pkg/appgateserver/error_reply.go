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
	err error,
) {
	responseBz, err := partials.GetErrorReply(ctx, payloadBz, err)
	if err != nil {
		app.logger.Error().Err(err).Msg("failed getting error reply")
		return
	}

	if _, err = writer.Write(responseBz); err != nil {
		app.logger.Error().Err(err).Msg("failed writing relay response")
		return
	}
}
