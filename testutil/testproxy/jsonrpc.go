package testproxy

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

// prepareJsonRPCResponsePayload prepares a hard-coded JsonRPC payload for a specific response.
func prepareJsonRPCResponsePayload() []byte {
	return []byte(`{"jsonrpc":"2.0","id":1,"result":"some result"}`)
}

// prepareJsonRPCResponsePayload prepares a hard-coded JsonRPC payload for a specific request.
func PrepareJsonRPCRequestPayload() []byte {
	return []byte(`{"method":"someMethod","id":1,"jsonrpc":"2.0","params":["someParam"]}`)
}
