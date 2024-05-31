package testproxy

import (
	"bytes"
	"io"
	"net/http"
	"net/url"

	"github.com/pokt-network/shannon-sdk/httpcodec"
)

// JSONRpcError is the error struct for the JSON RPC response
type JSONRpcError struct {
	Code    int32  `json:"code"`
	Message string `json:"message"`
}

// JSONRpcErrorReply is the error reply struct for the JSON RPC response
type JSONRpcErrorReply struct {
	Id      int32  `json:"id"`
	Jsonrpc string `json:"jsonrpc"`
	Error   *JSONRpcError
}

// prepareJsonRPCResponse prepares a hard-coded JsonRPC payload for a specific response.
func prepareJsonRPCResponse() []byte {
	bodyBz := []byte(`{"jsonrpc":"2.0","id":1,"result":"some result"}`)

	response := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		Body:       io.NopCloser(bytes.NewReader(bodyBz)),
	}
	response.Header.Set("Content-Type", "application/json")

	responseBz, _ := httpcodec.SerializeHTTPResponse(response)
	return responseBz
}

// PrepareJsonRPCRequest prepares a hard-coded JsonRPC payload for a specific request.
func PrepareJsonRPCRequest() []byte {
	bodyBz := []byte(`{"method":"someMethod","id":1,"jsonrpc":"2.0","params":["someParam"]}`)
	request := &http.Request{
		Method: http.MethodPost,
		URL:    &url.URL{},
		Header: http.Header{},
		Body:   io.NopCloser(bytes.NewReader(bodyBz)),
	}
	request.Header.Set("Content-Type", "application/json")

	requestBz, _ := httpcodec.SerializeHTTPRequest(request)
	return requestBz
}
