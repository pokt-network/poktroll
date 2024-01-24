package proxy

import (
	"context"
	"net/http"

	"github.com/pokt-network/poktroll/pkg/partials"
	"github.com/pokt-network/poktroll/x/service/types"
)

// replyWithError builds the appropriate error format according to the payload
// using the passed in error and writes it to the writer.
// NOTE: This method is used to reply with an "internal" error that is related
// to the proxy itself and not to the relayed request.
func (sync *synchronousRPCServer) replyWithError(
	ctx context.Context,
	payloadBz []byte,
	writer http.ResponseWriter,
	proxyName string,
	serviceId string,
	err error,
) {
	relaysErrorsTotal.With("service_id", serviceId, "proxy_name", proxyName).Add(1)

	responseBz, err := partials.GetErrorReply(ctx, payloadBz, err)
	if err != nil {
		sync.logger.Error().Err(err).Str("service_id", serviceId).Str("proxy_name", proxyName).Msg(
			"failed getting error reply")
		return
	}

	relayResponse := &types.RelayResponse{Payload: responseBz}

	relayResponseBz, err := relayResponse.Marshal()
	if err != nil {
		sync.logger.Error().Err(err).Str("service_id", serviceId).Str("proxy_name", proxyName).Msg(
			"failed marshaling relay response")
		return
	}

	if _, err = writer.Write(relayResponseBz); err != nil {
		sync.logger.Error().Err(err).Str("service_id", serviceId).Str("proxy_name", proxyName).Msg(
			"failed writing relay response")
		return
	}
}
