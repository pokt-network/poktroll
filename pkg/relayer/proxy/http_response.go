package proxy

import (
	"context"
	"net/http"
	"time"

	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/x/service/types"
)

// handleHttpStream processes streaming HTTP responses from backend services.
//
// Streaming flow:
//  1. Read each newline-delimited chunk from backend response
//  2. Wrap chunk in POKT HTTP response structure (status code, headers, body)
//  3. Sign each chunk individually using supplier's key
//  4. Write signed chunk with delimiter to client
//  5. Flush immediately to ensure low-latency streaming
//
// This enables real-time streaming for SSE and NDJSON responses while maintaining
// POKT's signature verification requirements.
//
// TODO_IMPROVE: Consider adding configurable buffer size for scanner to handle
// large streaming chunks (default is 64KB).
// Some LLM responses may exceed this.
//
// Returns:
//   - Final relay response (contains last chunk's signature)
//   - Total response size across all chunks (for metrics)
//   - Error if streaming fails (network errors, signature failures, etc.)
func (server *relayMinerHTTPServer) handleHttp(
	ctx context.Context,
	logger polylog.Logger,
	relayRequest *types.RelayRequest,
	response *http.Response,
	writer http.ResponseWriter,
) (*types.RelayResponse, float64, error) {
	backendServiceProcessingEnd := time.Now()

	// Extract the metadata from the relay request
	meta := relayRequest.Meta

	// Initialize empty relay response with metadata only
	relayResponse := &types.RelayResponse{
		Meta: types.RelayResponseMetadata{SessionHeader: meta.SessionHeader},
	}

	// Serialize backend response (status code + headers + body)
	serializedHTTPResponse, responseBz, err := SerializeHTTPResponse(logger, response, server.serverConfig.MaxBodySize)
	if err != nil {
		logger.Error().Err(err).Msg("❌ Failed serializing the service response")
		return nil, 0, err
	}

	// Close backend response body early to free connection pool resources
	CloseBody(logger, response.Body)

	// Pass through all backend responses including errors.
	// Allows clients to see real HTTP status codes from backend service.
	// Log non-2XX status codes for monitoring but don't block response.
	if response.StatusCode >= http.StatusMultipleChoices {
		logger.Error().
			Int("status_code", response.StatusCode).
			Str("request_payload_first_bytes", polylog.Preview(string(relayRequest.Payload))).
			Str("response_payload_first_bytes", polylog.Preview(string(serializedHTTPResponse.BodyBz))).
			Msg("backend service returned a non-2XX status code. Passing it through to the client.")
	}

	logger.Debug().
		Str("relay_request_session_header", meta.SessionHeader.String()).
		Msg("building relay response protobuf from service response")

	// Check context cancellation before building relay response to prevent signature race conditions
	if ctxErr := ctx.Err(); ctxErr != nil {
		logger.Warn().Err(ctxErr).Msg("⚠️ Context canceled before building relay response - preventing signature race condition")
		return nil, 0, ErrRelayerProxyTimeout.Wrapf(
			"request context canceled during response building: %v",
			ctxErr,
		)
	}

	// Build the relay response using the original service's response.
	relayResponse, err = server.newRelayResponse(responseBz, meta.SessionHeader, meta.SupplierOperatorAddress)
	if err != nil {
		logger.Error().Err(err).Msg("❌ Failed building the relay response")
		// The client should not have knowledge about the RelayMiner's issues with building the relay response.
		// Reply with an internal error so that the original error is not exposed to the client.
		return nil, 0, ErrRelayerProxyInternalError.Wrap(err.Error())
	}

	// Capture the time after response time for the relay.
	responsePreparationEnd := time.Now()

	// Add response preparation duration to the logger such that any log before errors will have
	// as much request duration information as possible.
	logger = logger.With(
		"response_preparation_duration",
		time.Since(backendServiceProcessingEnd).String(),
	)
	relayer.CaptureResponsePreparationDuration(meta.SessionHeader.ServiceId, backendServiceProcessingEnd)

	// Send the relay response to the client.
	err = server.sendRelayResponse(relayResponse, writer)
	logger = logger.With("send_response_duration", time.Since(responsePreparationEnd).String())
	if err != nil {
		clientError := ErrRelayerProxyInternalError.Wrap(err.Error())
		// Log current time to highlight writer i/o timeout errors.
		logger.Warn().Err(err).Time("current_time", time.Now()).Msg("❌ Failed sending relay response")
		return nil, 0, clientError
	}

	// Set response size
	responseSize := float64(relayResponse.Size())

	// Return the relay response, response size, and nil error.
	return relayResponse, responseSize, nil
}
