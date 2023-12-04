package testproxy

import "fmt"

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

func prepareJsonRPCPayload(serviceId string) string {
	return fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"result":"%s"}`, serviceId)
}
