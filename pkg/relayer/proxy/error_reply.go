package proxy

import (
	"errors"
	"net/http"

	"github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// replyWithError builds the appropriate error format according to the RelayRequest
// using the passed in error and writes it to the writer.
// NOTE: This method is used to reply with an "internal" error that is related
// to the server itself and not to the relayed request.
func (sync *synchronousRPCServer) replyWithError(
	replyError error,
	relayRequest *types.RelayRequest,
	writer http.ResponseWriter,
) {
	// Indicate whether the original error should be sent to the client or send
	// a generic error reply.
	if errors.Is(replyError, ErrRelayerProxyInternalError) {
		replyError = ErrRelayerProxyInternalError
	}
	listenAddress := sync.serverConfig.ListenAddress

	// Fill in the needed missing fields of the RelayRequest with empty values.
	relayRequest = mergeWithEmptyRelayRequest(relayRequest)
	serviceId := relayRequest.Meta.SessionHeader.Service.Id

	relaysErrorsTotal.With("service_id", serviceId).Add(1)

	// Create an unsigned RelayResponse with the error reply as payload and the
	// same session header as the source RelayRequest.
	relayResponse := &types.RelayResponse{
		Meta: types.RelayResponseMetadata{
			SessionHeader: relayRequest.Meta.SessionHeader,
		},
		Payload: []byte(replyError.Error()),
	}

	relayResponseBz, err := relayResponse.Marshal()
	if err != nil {
		sync.logger.Error().Err(err).
			Str("service_id", serviceId).
			Str("listen_address", listenAddress).
			Msg("failed marshaling error relay response")
		return
	}

	if _, err = writer.Write(relayResponseBz); err != nil {
		sync.logger.Error().Err(err).
			Str("service_id", serviceId).
			Str("listen_address", listenAddress).
			Msg("failed writing error relay response")
		return
	}
}

// mergeWithEmptyRelayRequest generates an empty RelayRequest that has the same
// service and payload as the source RelayRequest if they are not nil.
// It is meant to be used when replying with an error but no valid RelayRequest is available.
func mergeWithEmptyRelayRequest(sourceRelayRequest *types.RelayRequest) *types.RelayRequest {
	emptyRelayRequest := &types.RelayRequest{
		Meta: types.RelayRequestMetadata{
			SessionHeader: &sessiontypes.SessionHeader{
				Service: &sharedtypes.Service{
					Id: "",
				},
			},
		},
		Payload: []byte{},
	}

	if sourceRelayRequest == nil {
		return emptyRelayRequest
	}

	if sourceRelayRequest.Payload != nil {
		emptyRelayRequest.Payload = sourceRelayRequest.Payload
	}

	if sourceRelayRequest.Meta.SessionHeader != nil {
		if sourceRelayRequest.Meta.SessionHeader.Service != nil {
			emptyRelayRequest.Meta.SessionHeader.Service = sourceRelayRequest.Meta.SessionHeader.Service
		}
	}

	return emptyRelayRequest
}
