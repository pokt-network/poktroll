package proxy

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"

	sdktypes "github.com/pokt-network/shannon-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/pokt-network/poktroll/pkg/relayer/config"

	"github.com/pokt-network/poktroll/pkg/polylog"
)

const (
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

// CloseBody safely closes an io.ReadCloser with proper error handling and logging.
// It gracefully handles nil readers and logs any errors encountered during closure.
func CloseBody(logger polylog.Logger, body io.ReadCloser) {
	if body == nil {
		logger.Warn().Msg("⚠️ SHOULD NEVER HAPPEN ⚠️ Attempting to close nil request body")
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
func SafeReadBody(
	logger polylog.Logger,
	body io.ReadCloser,
	maxSize int64,
) (bodyBytes []byte, cleanupFunc func(), err error) {
	defer CloseBody(logger, body)

	if maxSize <= 0 {
		return nil, nil, config.ErrRelayMinerConfigInvalidMaxBodySize.Wrapf(
			"invalid max body size %q",
			maxSize,
		)
	}

	// Create a limited reader that will read at most maxSize+1 bytes
	// The +1 allows us to detect when the body exceeds the limit
	limitedReader := io.LimitReader(body, maxSize+1)

	// Get a reusable *bytes.Buffer from the pool
	buf := bodyBufPool.Get().(*bytes.Buffer)

	// Create a cleanup function that resets and returns the buffer to the pool.
	// This closure pattern is necessary because:
	// - The buffer contents (buf.Bytes()) are returned to the caller
	// - The buffer itself must be returned to the pool only after the caller has finished using the data
	// - The caller is responsible for calling this cleanup function when the buffer data is no longer needed
	// - This MUST be deferred to ensure any (un)marshalling is complete before releasing the buffer
	// - The cleanup function MUST be called only once to avoid double cleanup
	bufferCleanedUp := false
	resetReadBodyPoolBytes := func() {
		// Avoid double cleanup
		if bufferCleanedUp {
			return
		}
		buf.Reset() // Always reset before use
		bodyBufPool.Put(buf)
		// Mark as cleaned up to prevent double cleanup
		bufferCleanedUp = true
	}

	bytesRead, err := buf.ReadFrom(limitedReader)
	if err != nil {
		resetReadBodyPoolBytes()
		return nil, nil, ErrRelayerProxyInternalError.Wrapf(
			"failed to read request body: %s", err.Error(),
		)
	}

	// Check if the body exceeded our size limit
	if bytesRead > maxSize {
		resetReadBodyPoolBytes()
		return nil, nil, ErrRelayerProxyMaxBodyExceeded.Wrapf(
			"body size exceeds maximum allowed body: %d bytes read > %d bytes limit",
			bytesRead,
			maxSize,
		)
	}

	return buf.Bytes(), resetReadBodyPoolBytes, nil
}

// SafeRequestReadBody reads the HTTP request body up to a specified size limit, enforcing safety and logging errors.
// Logs and wraps errors for size violations or reading issues, using the provided logger. Returns body as []byte or error.
func SafeRequestReadBody(
	logger polylog.Logger,
	request *http.Request,
	maxSize int64,
) (bodyBytes []byte, cleanupFunc func(), err error) {
	body, resetReadBodyPoolBytes, err := SafeReadBody(logger, request.Body, maxSize)

	if errors.Is(err, ErrRelayerProxyMaxBodyExceeded) {
		return nil, nil, ErrRelayerProxyRequestLimitExceeded.Wrap(err.Error())
	}

	return body, resetReadBodyPoolBytes, err
}

// SafeResponseReadBody reads the HTTP response body up to a specified size limit, enforcing safety and logging errors.
// Logs and wraps errors for size violations or reading issues, using the provided logger. Returns body as []byte or error.
func SafeResponseReadBody(
	logger polylog.Logger,
	response *http.Response,
	maxSize int64,
) (bodyBytes []byte, cleanupFunc func(), err error) {
	body, resetReadBodyPoolBytes, err := SafeReadBody(logger, response.Body, maxSize)

	if errors.Is(err, ErrRelayerProxyMaxBodyExceeded) {
		return nil, resetReadBodyPoolBytes, ErrRelayerProxyResponseLimitExceeded.Wrap(err.Error())
	}

	return body, resetReadBodyPoolBytes, err
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
	maxBodySize int64,
) (poktHTTPResponse *sdktypes.POKTHTTPResponse, poktHTTPResponseBz []byte, err error) {
	// Read the response body with size limits
	responseBodyBz, resetReadBodyPoolBytes, err := SafeResponseReadBody(logger, response, maxBodySize)
	// Handle error case if SafeResponseReadBody fails:
	// - The buffer pool cleanup has already been performed internally
	// - We can return early without calling resetReadBodyPoolBytes
	// - resetReadBodyPoolBytes would be nil anyway in this case
	if err != nil {
		if resetReadBodyPoolBytes != nil {
			// Ensure buffer is returned to pool on error
			resetReadBodyPoolBytes()
		}
		return nil, nil, err
	}
	// Ensure buffer is returned to pool when function exits.
	// - responseBodyBz will no longer be needed when poktHTTPResponseBz is marshaled below.
	// - This MUST be deferred to ensure any (un)marshalling is complete before releasing the buffer
	defer resetReadBodyPoolBytes()

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
