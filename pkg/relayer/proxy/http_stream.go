package proxy

import (
	"bufio"
	"fmt"
	"net/http"
	"strings"

	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/x/service/types"
	sdktypes "github.com/pokt-network/shannon-sdk/types"
	"google.golang.org/protobuf/proto"
)

// Target streaming types
var httpStreamingTypes = []string{
	"text/event-stream",
	"application/x-ndjson",
}

const streamDelimitter = "||POKT_STREAM||"

// Custom split function for POKT events.
// The POKT streams contains a signature and the body of the request to the
// backend, so we need a custom delimiter for streaming responses.
// This functions is a custom delimiter for the Stream functionality
func ScanEvents(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	// Look for the POKT_STREAM delimiter
	if i := strings.Index(string(data), streamDelimitter); i >= 0 {
		// Return chunk without the delimiter
		return i + len(streamDelimitter), data[0:i], nil
	}

	// If we're at EOF, return whatever we have
	if atEOF {
		return len(data), data, nil
	}

	// Request more data
	return 0, nil, nil
}

// This function looks for the supported streams.
// We should be able to handle any cunked stream, but we have limited this
// to the content types specified in the variable `httpStreamingTypes`
//
// Returns:
//   - boolean whether the backend response should be streamed or not
func IsStreamingResponse(response *http.Response) bool {
	// Check if this is a streaming response
	contentType := strings.ToLower(response.Header.Get("Content-Type"))
	for _, streamType := range httpStreamingTypes {
		if strings.Contains(contentType, streamType) {
			return true
		}
	}
	return false
}

// Handles the streaming backend responses.
// This function takes a streaming body and:
// - Reads each chunk
// - Converts the chunk into a POKT HTTP response structure
// - Signs each chunk
// - Writes and flush to the requesting app
//
// This will handle any stream where the delimiter is a newline (`\n`)
func (server *relayMinerHTTPServer) HandleHttpStream(
	response *http.Response,
	writer http.ResponseWriter,
	meta types.RelayRequestMetadata,
	logger polylog.Logger,
) (relayResponse *types.RelayResponse, responseSize float64, err error) {
	// Set headers
	writer.Header().Set("Connection", "close")
	// Copy the response back to the original request
	for k, v := range response.Header {
		writer.Header()[k] = v
	}
	writer.WriteHeader(response.StatusCode)

	// Instance the flusher
	flusher, ok := writer.(http.Flusher)
	if !ok {
		logger.Error().Msg("Streaming not supported.")
		return nil, 0, fmt.Errorf("❌ failed to open stream request")
	}

	// Create scanner with default delimiter (\n)
	scanner := bufio.NewScanner(response.Body)
	// Scan for chunks
	for scanner.Scan() {
		line := scanner.Bytes()
		// Add back the newline that we stripped at Scan
		line = append(line, '\n')

		// Create the POKT HTTP response structure
		poktHTTPResponse := &sdktypes.POKTHTTPResponse{
			StatusCode: uint32(http.StatusOK),
			Header:     make(map[string]*sdktypes.Header, 0),
			BodyBz:     line,
		}
		// Deterministic marshaling of response
		marshalOpts := proto.MarshalOptions{Deterministic: true}
		poktHTTPResponseBz, err := marshalOpts.Marshal(poktHTTPResponse)
		if err != nil {
			return nil, 0, fmt.Errorf("❌ failed to marshal POKT HTTP response: %w", err)
		}

		// Sign response
		relayResponse, err = server.newRelayResponse(poktHTTPResponseBz, meta.SessionHeader, meta.SupplierOperatorAddress)
		if err != nil {
			return nil, 0, err
		}
		// Marshal response
		signedLine, err := relayResponse.Marshal()
		if err != nil {
			return nil, 0, err
		}

		// track size, the sum of all chunks
		responseSize += float64(relayResponse.Size())

		// Append custom delimiter (used by app to detect POKT streaming)
		signedLine = append(signedLine, []byte(streamDelimitter)...)

		// Write to client
		_, err = writer.Write(signedLine)
		if err != nil {
			return nil, 0, err
		}

		// Flush to ensure the stream goes asap to the app
		flusher.Flush()
	}

	return
}
