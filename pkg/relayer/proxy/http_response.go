package proxy

import (
	"context"
	"net/http"
	"time"

	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/x/service/types"
)

// handleHttp builds and sends a signed POKT relay response from a backend HTTP response.
//
// Flow:
//  1. Record backend processing end time for metrics.
//  2. Extract request metadata; initialize an empty RelayResponse with SessionHeader.
//  3. Serialize backend response (status, headers, body) with MaxBodySize limit.
//  4. Close backend response body to free resources.
//  5. If status >= 300, log details and pass through (do not block).
//  6. Check context cancellation to avoid signature race conditions.
//  7. Build the signed relay response via newRelayResponse(...).
//  8. Capture preparation timing, annotate logger, and record metrics via
//     relayer.CaptureResponsePreparationDuration.
//  9. Send the relay response to the client via sendRelayResponse(...); map
//     any error to an internal error and return.
// 10. Compute and return the relay response and its size.
//
// Notes:
// - This path handles full HTTP responses, not streaming chunked signing.
// - For streaming/SSE/NDJSON, use dedicated streaming handlers.
//
// Returns:
//  - Final relay response.
//  - Total response size (bytes) for metrics.
//  - Error if serialization, signing, or write fails.
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
