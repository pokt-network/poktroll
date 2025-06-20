package proxy

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/pokt-network/poktroll/pkg/polylog"
	"google.golang.org/protobuf/proto"

	sdktypes "github.com/pokt-network/shannon-sdk/types"
)

// TODO(@jorgecuesta) Move this into `config.yaml` as a default AND/OR per service this is important avoid anyone could blow a RM due to bigger payload attack.
// DefaultMaxBodySize - Defines max request/response body to be handled
const DefaultMaxBodySize = 20 * 1024 * 1024 // 20MB

// bodyBufPool - Pool with 8MB slices, efficient for most realistic body sizes
var bodyBufPool = sync.Pool{
	New: func() any {
		return make([]byte, 0, 8*1024*1024) // 2MB pre allocated
	},
}

// CloseRequestBody safely closes an io.ReadCloser, logging a warning if it's nil or an error if closing fails.
func CloseRequestBody(logger polylog.Logger, body io.ReadCloser) {
	if body == nil {
		logger.Warn().Msg("⚠️ SHOULD NEVER HAPPEN ⚠️ Attempting to close request body when it is nil.")
		return
	}
	e := body.Close()
	if e != nil {
		logger.Error().Err(e).Msg("❌ failed to close the request body")
	}
}

// SafeReadBody reads the full body from the reader with a max size limit.
// It closes the reader automatically and reuses memory from a buffer pool.
func SafeReadBody(body io.ReadCloser, maxSize int64, logger polylog.Logger) ([]byte, error) {
	defer CloseRequestBody(logger, body)

	if maxSize <= 0 {
		maxSize = DefaultMaxBodySize
	}

	// LimitReader to cap how much we read
	limited := io.LimitReader(body, maxSize+1)

	// Get a buffer from the pool and wrap in bytes.Buffer
	poolBuf := bodyBufPool.Get().([]byte)
	defer bodyBufPool.Put(poolBuf[:0]) // Reset before returning to the pool

	buf := bytes.NewBuffer(poolBuf[:0])
	n, err := buf.ReadFrom(limited)
	if err != nil {
		return nil, err
	}
	if n > maxSize {
		e := fmt.Errorf("body larger than max size: %d > %d", n, maxSize)
		logger.Error().Err(e).Msg("❌ Request/Response body larger than max size.")
		return nil, e
	}

	return buf.Bytes(), nil
}

// NOTE: I move this from Shannon-SDK because it was not closing body properly and also using the io.ReadAll
// SerializeHTTPResponse take an http.Response object and serializes it into a byte
// slice that can be embedded into another struct, such as RelayResponse.Payload.
func SerializeHTTPResponse(
	response *http.Response,
	logger polylog.Logger,
) (poktHTTPResponse *sdktypes.POKTHTTPResponse, poktHTTPResponseBz []byte, err error) {
	responseBodyBz, err := SafeReadBody(response.Body, DefaultMaxBodySize, logger)
	if err != nil {
		return nil, nil, err
	}
	defer CloseRequestBody(logger, response.Body)

	headers := map[string]*sdktypes.Header{}
	// http.Header is a map of header keys to a list of values. We need to get
	// the http.Header.Values(key) to get all the values of a key.
	// We have to avoid using http.Header.Get(key) because it only returns the
	// first value of the key.
	for key := range response.Header {
		headerValues := response.Header.Values(key)
		headers[key] = &sdktypes.Header{
			Key:    key,
			Values: headerValues,
		}
	}

	poktHTTPResponse = &sdktypes.POKTHTTPResponse{
		StatusCode: uint32(response.StatusCode),
		Header:     headers,
		BodyBz:     responseBodyBz,
	}

	// Use deterministic marshaling to ensure that the serialized response is
	// byte-for-byte equal when comparing the serialized response.
	opts := proto.MarshalOptions{Deterministic: true}

	poktHTTPResponseBz, err = opts.Marshal(poktHTTPResponse)

	return poktHTTPResponse, poktHTTPResponseBz, err
}
