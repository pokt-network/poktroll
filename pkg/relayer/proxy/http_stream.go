package proxy

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"strings"

	sdktypes "github.com/pokt-network/shannon-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/x/service/types"
)

// Target streaming types
var httpStreamingTypes = []string{
	"text/event-stream",
	"application/x-ndjson",
}

const streamDelimiter = "||POKT_STREAM||"

// ScanEvents is a custom bufio.SplitFunc for scanning POKT streaming responses.
//
// POKT streams contain a signature and the body of the request to the backend,
// requiring a custom delimiter (||POKT_STREAM||) instead of standard newlines.
//
// This function splits incoming stream data by the POKT_STREAM delimiter,
// allowing clients to parse individual signed chunks from the relay response.
func ScanEvents(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	// Look for the POKT_STREAM delimiter
	delimiterBz := []byte(streamDelimiter)
	if i := bytes.Index(data, delimiterBz); i >= 0 {
		// Return chunk without the delimiter
		return i + len(delimiterBz), data[0:i], nil
	}

	// If we're at EOF, return whatever we have
	if atEOF {
		return len(data), data, nil
	}

	// Request more data
	return 0, nil, nil
}

// isStreamingResponse determines if an HTTP response should be handled as a stream.
//
// Checks the Content-Type header against supported streaming types:
//   - text/event-stream (Server-Sent Events)
//   - application/x-ndjson (Newline-Delimited JSON)
//
// Note: While we could handle any chunked stream, we limit support to these
// content types to ensure predictable behavior and proper testing coverage.
//
// Returns true if the response should be streamed chunk-by-chunk with
// individual signatures per chunk.
func isStreamingResponse(response *http.Response) bool {
	contentType := strings.ToLower(response.Header.Get("Content-Type"))
	for _, streamType := range httpStreamingTypes {
		if strings.Contains(contentType, streamType) {
			return true
		}
	}
	return false
}

// HandleHttpStream processes streaming HTTP responses from backend services.
//
// Streaming flow:
//  1. Read each newline-delimited chunk from backend response
//  2. Wrap chunk in POKT HTTP response structure (status code, headers, body)
//  3. Sign each chunk individually using supplier's key
//  4. Write signed chunk with POKT_STREAM delimiter to client
//  5. Flush immediately to ensure low-latency streaming
//
// This enables real-time streaming for SSE and NDJSON responses while maintaining
// POKT's signature verification requirements.
//
// Note: Only handles streams with newline (\n) delimiters. Other delimiters
// require custom scanner configuration.
//
// TODO_IMPROVE: Consider adding configurable buffer size for scanner to handle
// large streaming chunks (default is 64KB). Some LLM responses may exceed this.
//
// Returns:
//   - Final relay response (contains last chunk's signature)
//   - Total response size across all chunks (for metrics)
//   - Error if streaming fails (network errors, signature failures, etc.)
func (server *relayMinerHTTPServer) HandleHttpStream(
	response *http.Response,
	writer http.ResponseWriter,
	meta types.RelayRequestMetadata,
	logger polylog.Logger,
) (relayResponse *types.RelayResponse, responseSize float64, err error) {
	// Ensure response body is closed when streaming completes or fails
	defer CloseBody(logger, response.Body)

	// Copy all backend headers to client response
	for k, v := range response.Header {
		writer.Header()[k] = v
	}
	// Force connection close to prevent client reuse issues with streaming
	writer.Header().Set("Connection", "close")
	writer.WriteHeader(response.StatusCode)

	// Verify writer supports flushing (required for streaming)
	flusher, ok := writer.(http.Flusher)
	if !ok {
		logger.Error().Msg("Streaming not supported - ResponseWriter does not implement http.Flusher")
		return nil, 0, fmt.Errorf("❌ failed to open stream request: flusher unavailable")
	}

	// Create scanner with default newline delimiter
	scanner := bufio.NewScanner(response.Body)

	// Process each chunk from backend stream
	for scanner.Scan() {
		line := scanner.Bytes()
		// Restore newline stripped by scanner (needed for protocol compatibility)
		line = append(line, '\n')

		// Wrap chunk in POKT HTTP response structure
		poktHTTPResponse := &sdktypes.POKTHTTPResponse{
			StatusCode: uint32(http.StatusOK),
			Header:     make(map[string]*sdktypes.Header, 0),
			BodyBz:     line,
		}

		// Marshal with deterministic ordering for signature consistency
		marshalOpts := proto.MarshalOptions{Deterministic: true}
		poktHTTPResponseBz, err := marshalOpts.Marshal(poktHTTPResponse)
		if err != nil {
			return nil, 0, fmt.Errorf("❌ failed to marshal POKT HTTP response: %w", err)
		}

		// Sign this chunk
		relayResponse, err = server.newRelayResponse(poktHTTPResponseBz, meta.SessionHeader, meta.SupplierOperatorAddress)
		if err != nil {
			return nil, 0, fmt.Errorf("❌ failed to sign relay response chunk: %w", err)
		}

		// Serialize signed response
		signedLine, err := relayResponse.Marshal()
		if err != nil {
			return nil, 0, fmt.Errorf("❌ failed to marshal signed relay response: %w", err)
		}

		// Track cumulative size across all chunks
		responseSize += float64(relayResponse.Size())

		// Append POKT stream delimiter (allows client-side chunk detection)
		signedLine = append(signedLine, []byte(streamDelimiter)...)

		// Write signed chunk to client
		if _, err = writer.Write(signedLine); err != nil {
			return nil, 0, fmt.Errorf("❌ failed to write stream chunk to client: %w", err)
		}

		// Flush immediately for low-latency streaming
		flusher.Flush()
	}

	// Check for scanner errors (network issues, buffer overflows, etc.)
	if err := scanner.Err(); err != nil {
		return nil, 0, fmt.Errorf("❌ stream scanning error: %w", err)
	}

	return
}
