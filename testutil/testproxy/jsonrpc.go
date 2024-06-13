package testproxy

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"testing"

	sdktypes "github.com/pokt-network/shannon-sdk/types"
	"github.com/stretchr/testify/require"
)

// JSONRPCError is the error struct for the JSONRPC response payload.
type JSONRPCError struct {
	Code    int32  `json:"code"`
	Message string `json:"message"`
}

// JSONRPCErrorReply is the error reply struct for the JSON-RPC response payload.
type JSONRPCErrorReply struct {
	Id      int32  `json:"id"`
	Jsonrpc string `json:"jsonrpc"`
	Error   *JSONRPCError
}

// sendJSONRPCResponse constructs a hard-coded JSON-RPC response and writes
// it to the provided http.ResponseWriter.
//
// It uses a default StatusOK and "application/json" content type, along with
// the provided hard-coded body bytes.
//
// Unlike PrepareJSONRPCRequest, this function is NOT EXPORTED as it is
// exclusively used within the testutil/testproxy package for serving a
// hard-coded JSON-RPC response.
//
// IMPORTANT: This function is intended solely for testing purposes and
// SHOULD NOT be used in production code.
func sendJSONRPCResponse(t *testing.T, w http.ResponseWriter) {
	t.Helper()
	bodyBz := []byte(`{"jsonrpc":"2.0","id":1,"result":"some result"}`)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write(bodyBz)
	require.NoError(t, err)
}

// PrepareJSONRPCRequest constructs a hard-coded JSON-RPC http.Request and
// returns the corresponding sdk serialized POKTHTTPRequest.
//
// It uses the default POST method and "application/json" content type, along
// with the provided hard-coded body bytes.
// The function then serializes the entire generated http.Request into an sdk
// serialized POKTHTTPRequest to be embedded in a RelayRequest.Payload.
//
// Unlike sendJSONRPCResponse, this function IS EXPORTED to be used in
// the pkg/relayer/proxy/proxy_test testing code.
//
// IMPORTANT: This function is intended solely for testing purposes and
// SHOULD NOT be used in production code.
func PrepareJSONRPCRequest(t *testing.T) []byte {
	t.Helper()
	bodyBz := []byte(`{"method":"someMethod","id":1,"jsonrpc":"2.0","params":["someParam"]}`)
	request := &http.Request{
		Method: http.MethodPost,
		URL:    &url.URL{},
		Header: http.Header{},
		Body:   io.NopCloser(bytes.NewReader(bodyBz)),
	}
	request.Header.Set("Content-Type", "application/json")

	_, requestBz, err := sdktypes.SerializeHTTPRequest(request)
	require.NoError(t, err)
	return requestBz
}
