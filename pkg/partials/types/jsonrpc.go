package types

import (
	"encoding/json"

	"github.com/pokt-network/poktroll/x/shared/types"
)

// PartialJSONPayload is a partial representation of a JSON-RPC request payload
// that contains only the fields necessary to generate an error response and
// handle accounting for the request's method field.
type PartialJSONPayload struct {
	Id      uint64 `json:"id"`
	JsonRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
}

// GetRequestType returns the request type for the given payload.
func (j *PartialJSONPayload) GetRPCType() types.RPCType {
	return types.RPCType_JSON_RPC
}

// GenerateErrorPayload creates a JSON-RPC error payload from the provided
// error with the macthing json-rpc and id fields from the request payload.
func (j *PartialJSONPayload) GenerateErrorPayload(err error) ([]byte, error) {
	reply := map[string]any{
		"jsonrpc": j.JsonRPC,
		"id":      j.Id,
		"error": map[string]any{
			"code":    -32000,
			"message": err.Error(),
			"data":    nil,
		},
	}
	return json.Marshal(reply)
}

// GetMethodWeighting returns the weight of the request's method field.
func (j *PartialJSONPayload) GetMethodWeighting() (uint64, error) {
	// TODO(@h5law): Implement this method
	return 0, nil
}
