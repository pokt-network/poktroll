package proxy

import (
	"errors"
	"fmt"
	"net/http"

	sdkerrors "cosmossdk.io/errors"

	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

// wrappedError is an interface for errors that has been wrapped using Error.Wrap
// and can be unwrapped to reveal the underlying cause.
type wrappedError interface {
	Unwrap() error
}

// sdkError is an interface for the RelayMiner components' registered errors.
// It exposes methods to retrieve the codespace, error code, and error message.
type sdkError interface {
	Codespace() string
	Error() string
	ABCICode() uint32
}

// replyWithError builds the appropriate error format according to the RelayRequest
// using the passed in error and writes it to the writer.
// NOTE: This method is used to reply with an "internal" error that is related
// to the server itself and not to the relayed request.
func (sync *relayMinerHTTPServer) replyWithError(
	replyError error,
	relayRequest *types.RelayRequest,
	writer http.ResponseWriter,
) {
	// Initialize a serializable (empty) RelayRequest if the one provided is nil.
	if relayRequest == nil {
		relayRequest = &types.RelayRequest{
			Meta: types.RelayRequestMetadata{
				SessionHeader: &sessiontypes.SessionHeader{},
			},
		}
	}

	// Ensure the session header is initialized to a proper structure.
	if relayRequest.Meta.SessionHeader == nil {
		relayRequest.Meta.SessionHeader = &sessiontypes.SessionHeader{}
	}

	listenAddress := sync.serverConfig.ListenAddress
	serviceId := relayRequest.Meta.SessionHeader.ServiceId

	errorLogger := sync.logger.With(
		"service_id", serviceId,
		"listen_address", listenAddress,
	).Error()

	// Indicate whether the original error should be sent to the client or send a generic error reply.
	if errors.Is(replyError, ErrRelayerProxyInternalError) {
		// TODO_TECHDEBT: Reenable internal error obfuscation once we are confident
		// that the relayer proxy is stable w.r.t. protocol compliance.
		// replyError = ErrRelayerProxyInternalError
		errorLogger.Err(replyError).Msgf("⚠️ Temporarily Overriding %v error until the RelayMiner is stable w.r.t. protocol compliance", ErrRelayerProxyInternalError)
	}

	// Fill in the needed missing fields of the RelayRequest with empty values.
	relayer.RelaysErrorsTotal.With("service_id", serviceId).Add(1)

	// Create an unsigned RelayResponse with the error reply as payload and the
	// same session header as the source RelayRequest.
	relayResponse := &types.RelayResponse{
		Meta: types.RelayResponseMetadata{
			SessionHeader: relayRequest.Meta.SessionHeader,
			// The supplier does not sign the error response, so we leave the signature empty.
		},
		// TODO_FOLLOWUP: Send an empty payload once PATH supports reading errors form RelayMinerError.
		Payload:         []byte(replyError.Error()),
		RelayMinerError: unpackSDKError(replyError),
	}

	relayResponseBz, err := relayResponse.Marshal()
	if err != nil {
		errorLogger.Err(err).Msg("failed marshaling error relay response")
		return
	}

	relayResponseBzLenStr := fmt.Sprintf("%d", len(relayResponseBz))
	writer.Header().Set("Content-Length", relayResponseBzLenStr)
	writer.Header().Set("Connection", "close")
	if _, err = writer.Write(relayResponseBz); err != nil {
		errorLogger.Err(err).Msg("failed writing error relay response")
		return
	}
}

// TODO_TECHDEBT(@red-0ne): Revisit all the RelayMiner's returned errors and ensure:
//   - It is always returning a registered error (i.e. implements the sdkError interface and not a fmt.Errorf)
//   - All registered errors have meaningful and short description, the wrapping error will provide more context
//   - The errors belong to the right codespace
//
// unpackSDKError attempts to extract an sdkError from the provided error chain.
//
//   - If srcError is nil, it returns nil.
//   - If srcError or any error in its unwrap chain implements the sdkError interface,
//     it returns a RelayMinerError constructed from that sdkError.
//   - If no sdkError is found after unwrapping, it returns a default RelayMinerError
//     indicating an unrecognized error while preserving the original error message.
func unpackSDKError(srcError error) *types.RelayMinerError {
	// If srcError is nil, return nil
	if srcError == nil {
		return nil
	}

	errorMessage := srcError.Error()
	currentError := srcError

	// Create a default RelayMinerError to return if no sdkError is found
	defaultError := &types.RelayMinerError{
		Codespace:   sdkerrors.UndefinedCodespace,
		Description: "Unregistered error",
		Message:     errorMessage,
	}

	// Unwrap srcError until we find an sdkError
	for {
		// If srcError casts to sdkError, return it
		if sdkErr, ok := currentError.(sdkError); ok {
			return &types.RelayMinerError{
				Codespace:   sdkErr.Codespace(),
				Code:        sdkErr.ABCICode(),
				Description: sdkErr.Error(),
				Message:     errorMessage,
			}
		}

		// Try to unwrap the error
		wrapper, ok := currentError.(wrappedError)
		if !ok {
			// Can't unwrap further but no sdkError found, return default error
			return defaultError
		}

		// Unwrap and check for nil
		if currentError = wrapper.Unwrap(); currentError == nil {
			return defaultError
		}
	}
}
