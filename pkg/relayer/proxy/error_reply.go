package proxy

import (
	"errors"
	"net/http"

	"github.com/pokt-network/poktroll/proto/types/service"
)

// replyWithError builds the appropriate error format according to the RelayRequest
// using the passed in error and writes it to the writer.
// NOTE: This method is used to reply with an "internal" error that is related
// to the server itself and not to the relayed request.
func (sync *synchronousRPCServer) replyWithError(
	replyError error,
	relayRequest *service.RelayRequest,
	writer http.ResponseWriter,
) {
	// Indicate whether the original error should be sent to the client or send
	// a generic error reply.
	if errors.Is(replyError, ErrRelayerProxyInternalError) {
		replyError = ErrRelayerProxyInternalError
	}
	listenAddress := sync.serverConfig.ListenAddress

	// Fill in the needed missing fields of the RelayRequest with empty values.
	relayRequest = relayRequest.NullifyForObservability()
	serviceId := relayRequest.Meta.SessionHeader.Service.Id

	errorLogger := sync.logger.With().
		Error().
		Str("service_id", serviceId).
		Str("listen_address", listenAddress)

	relaysErrorsTotal.With("service_id", serviceId).Add(1)

	// Create an unsigned RelayResponse with the error reply as payload and the
	// same session header as the source RelayRequest.
	relayResponse := &service.RelayResponse{
		Meta: service.RelayResponseMetadata{
			SessionHeader: relayRequest.Meta.SessionHeader,
		},
		Payload: []byte(replyError.Error()),
	}

	relayResponseBz, err := relayResponse.Marshal()
	if err != nil {
		errorLogger.Err(err).Msg("failed marshaling error relay response")
		return
	}

	if _, err = writer.Write(relayResponseBz); err != nil {
		errorLogger.Err(err).Msg("failed writing error relay response")
		return
	}
}
