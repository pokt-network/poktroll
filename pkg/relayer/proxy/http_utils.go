package proxy

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync"

	sdktypes "github.com/pokt-network/shannon-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/pokt-network/poktroll/pkg/polylog"
)

const (
	// TODO_IMPROVE: Make this a RelayMiner config parameter
	// TODO_IMPROVE: Make this configurable on a per-service basis
	defaultMaxBodySize = 20 * 1024 * 1024 // 20MB max request/response body size

	// Buffer pool allocation size - 8MB should handle most realistic body sizes efficiently
	bufferPoolSize = 8 * 1024 * 1024 // 8MB
)

// bodyBufPool provides reusable *bytes.Buffer to reduce memory allocations
// when reading HTTP request/response bodies
var bodyBufPool = sync.Pool{
	New: func() any {
		buf := bytes.NewBuffer(make([]byte, 0, bufferPoolSize))
		return buf
	},
}

// CloseRequestBody safely closes an io.ReadCloser with proper error handling and logging.
// It gracefully handles nil readers and logs any errors encountered during closure.
func CloseRequestBody(logger polylog.Logger, body io.ReadCloser) {
	if body == nil {
		logger.Warn().Msg("⚠️  SHOULD NEVER HAPPEN ⚠️  Attempting to close nil request body")
		return
	}

	if err := body.Close(); err != nil {
		logger.Error().Err(err).Msg("❌ Failed to close request body")
	}
}

// SafeReadBody reads the complete body from an io.ReadCloser with size limits and memory pooling.
// It automatically closes the reader, enforces maximum size constraints, and reuses buffers
// from a shared pool to minimize memory allocations.
//
// Parameters:
//   - body: The io.ReadCloser to read from (will be closed automatically)
//   - maxSize: Maximum allowed body size in bytes (uses defaultMaxBodySize if <= 0)
//   - logger: Logger for error reporting
//
// Returns the complete body as a byte slice or an error if reading fails or size limit is exceeded.
func SafeReadBody(logger polylog.Logger, body io.ReadCloser, maxSize int64) ([]byte, error) {
	defer CloseRequestBody(logger, body)

	if maxSize <= 0 {
		logger.Warn().Msgf("SHOULD NOT HAPPEN: Max body size is less than or equal to 0, using default max body size of %d", defaultMaxBodySize)
		maxSize = defaultMaxBodySize
	}

	// Create a limited reader that will read at most maxSize+1 bytes
	// The +1 allows us to detect when the body exceeds the limit
	limitedReader := io.LimitReader(body, maxSize+1)

	// Get a reusable *bytes.Buffer from the pool
	buf := bodyBufPool.Get().(*bytes.Buffer)
	buf.Reset() // Always reset before use
	defer bodyBufPool.Put(buf)

	bytesRead, err := buf.ReadFrom(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("❌ failed to read request body: %w", err)
	}

	// Check if the body exceeded our size limit
	if bytesRead > maxSize {
		err := fmt.Errorf("❌ request body size exceeds maximum allowed body: %d bytes read > %d bytes limit", bytesRead, maxSize)
		logger.Error().Err(err).Msg("❌ Request/response body too large")
		return nil, err
	}

	return buf.Bytes(), nil
}

// TODO_TECHDEBT: Move this function back to the Shannon SDK. It was moved:
// 1. To ensure proper body closure
// 2. To avoid using io.ReadAll which doesn't implement size limits.
// 3. To iterate faster
//
// SerializeHTTPResponse converts an http.Response into a protobuf-serialized byte slice.
//
// The function:
//   - Safely reads the response body with size limits
//   - Preserves all HTTP headers (including multiple values per header key)
//   - Uses deterministic protobuf marshaling for consistent serialization
//   - Properly closes the response body
//
// Parameters:
//   - response: The HTTP response to serialize
//   - logger: Logger for error reporting
//
// Returns:
//   - poktHTTPResponse: The structured response object
//   - poktHTTPResponseBz: The serialized response as bytes
//   - err: Any error encountered during processing
func SerializeHTTPResponse(
	logger polylog.Logger,
	response *http.Response,
) (poktHTTPResponse *sdktypes.POKTHTTPResponse, poktHTTPResponseBz []byte, err error) {
	// Read the response body with size limits
	responseBodyBz, err := SafeReadBody(logger, response.Body, defaultMaxBodySize)
	if err != nil {
		return nil, nil, fmt.Errorf("❌ failed to read response body: %w", err)
	}

	// Convert HTTP headers to the POKT header format
	// Note: We use Values() instead of Get() to preserve all header values,
	// since HTTP allows multiple values for the same header key
	headers := make(map[string]*sdktypes.Header, len(response.Header))
	for headerKey := range response.Header {
		headerValues := response.Header.Values(headerKey)
		headers[headerKey] = &sdktypes.Header{
			Key:    headerKey,
			Values: headerValues,
		}
	}

	// Create the POKT HTTP response structure
	poktHTTPResponse = &sdktypes.POKTHTTPResponse{
		StatusCode: uint32(response.StatusCode),
		Header:     headers,
		BodyBz:     responseBodyBz,
	}

	// Use deterministic marshaling to ensure consistent byte-for-byte serialization
	// This is crucial for consensus mechanisms that rely on deterministic hashing
	marshalOpts := proto.MarshalOptions{Deterministic: true}
	poktHTTPResponseBz, err = marshalOpts.Marshal(poktHTTPResponse)
	if err != nil {
		return nil, nil, fmt.Errorf("❌ failed to marshal POKT HTTP response: %w", err)
	}

	return poktHTTPResponse, poktHTTPResponseBz, nil
}
