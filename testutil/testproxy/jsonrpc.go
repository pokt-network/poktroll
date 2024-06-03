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

// prepareJSONRPCResponse constructs a hard-coded JSON-RPC http.Response and
// returns the corresponding sdk serialized POKTHTTPResponse.
// It uses a default StatusOK and "application/json" content type, along with
// the provided hard-coded body bytes.
// The function then serializes the entire generated http.Response into an sdk
// serialized POKTHTTPResponse to be embedded in a RelayResponse.Payload.
// Unlike PrepareJSONRPCRequest, this function is not exported as it is
// exclusively used within the testutil/testproxy package for serving a
// hard-coded JSON-RPC response.
// This function is intended solely for testing purposes and should not be used
// in production code.
func prepareJSONRPCResponse(t *testing.T) []byte {
	t.Helper()
	bodyBz := []byte(`{"jsonrpc":"2.0","id":1,"result":"some result"}`)

	response := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		Body:       io.NopCloser(bytes.NewReader(bodyBz)),
	}
	response.Header.Set("Content-Type", "application/json")

	responseBz, err := sdktypes.SerializeHTTPResponse(response)
	require.NoError(t, err)
	return responseBz
}

// PrepareJSONRPCRequest constructs a hard-coded JSON-RPC http.Request and
// returns the corresponding sdk serialized POKTHTTPRequest.
// It uses the default POST method and "application/json" content type, along
// with the provided hard-coded body bytes.
// The function then serializes the entire generated http.Request into an sdk
// serialized POKTHTTPRequest to be embedded in a RelayRequest.Payload.
// Unlike prepareJSONRPCResponse, this function is exported to be used in
// the pkg/relayer/proxy/proxy_test testing code.
// This function is intended solely for testing purposes and should not be used
// in production code.
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

	requestBz, err := sdktypes.SerializeHTTPRequest(request)
	require.NoError(t, err)
	return requestBz
}
