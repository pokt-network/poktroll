package proxy

import (
	"net/http"

	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

// replyWithError builds the appropriate error format according to the RelayRequest
// using the passed in error and writes it to the writer.
// NOTE: This method is used to reply with an "internal" error that is related
// to the server itself and not to the relayed request.
func (sync *relayMinerHTTPServer) replyWithError(
	replyError error,
	relayRequest *types.RelayRequest,
	writer http.ResponseWriter,
) {
	// Indicate whether the original error should be sent to the client or send
	// a generic error reply.
	// TODO_TECHDEBT: Reenable internal error obfuscation once we are confident
	// that the relayer proxy is stable w.r.t. protocol compliance.
	// if errors.Is(replyError, ErrRelayerProxyInternalError) {
	// 	replyError = ErrRelayerProxyInternalError
	// }
	listenAddress := sync.serverConfig.ListenAddress

	// Initialize a generic RelayRequest if the one provided is nil.
	if relayRequest == nil {
		relayRequest = &types.RelayRequest{
			Meta: types.RelayRequestMetadata{
				SessionHeader: &sessiontypes.SessionHeader{},
			},
		}
	}

	// Ensure the session header is initialized.
	if relayRequest.Meta.SessionHeader == nil {
		relayRequest.Meta.SessionHeader = &sessiontypes.SessionHeader{}
	}

	// Fill in the needed missing fields of the RelayRequest with empty values.
	serviceId := relayRequest.Meta.SessionHeader.ServiceId

	errorLogger := sync.logger.With(
		"service_id", serviceId,
		"listen_address", listenAddress,
	).Error()

	relayer.RelaysErrorsTotal.With("service_id", serviceId).Add(1)

	// Create an unsigned RelayResponse with the error reply as payload and the
	// same session header as the source RelayRequest.
	relayResponse := &types.RelayResponse{
		Meta: types.RelayResponseMetadata{
			SessionHeader: relayRequest.Meta.SessionHeader,
			// The supplier does not sign the error response, so we leave the signature empty.
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
