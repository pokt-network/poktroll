package partials

import (
	"encoding/json"
)

var (
	_ PartialPayload = (*partialJSONPayload)(nil)
	_ PartialPayload = (*partialRESTPayload)(nil)
)

// partialJSONPayload is a partial representation of a JSON-RPC request payload
// that contains only the fields necessary to generate an error response and
// handle accounting for the request's method field.
type partialJSONPayload struct {
	Id      uint64 `json:"id"`
	JsonRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
}

// GenerateErrorPayload creates a JSON-RPC error payload from the provided
// error with the macthing json-rpc and id fields from the request payload.
func (j *partialJSONPayload) GenerateErrorPayload(err error) ([]byte, error) {
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
func (j *partialJSONPayload) GetMethodWeighting() (uint64, error) {
	// TODO(@h5law): Implement this method
	return 0, nil
}

// partialRESTPayload is a partial representation of a REST request payload
// that contains only the fields necessary to generate an error response and
// handle accounting for the request's method as determined by its headers.
type partialRESTPayload struct {
	Headers map[string]string `json:"headers"`
}

// GenerateErrorPayload creates a REST error payload using the headers from the
// request payload.
func (r *partialRESTPayload) GenerateErrorPayload(err error) ([]byte, error) {
	// TODO(@h5law): Implement this method
	return nil, nil
}

// GetMethodWeighting returns the weight of the request's method from the headers.
func (r *partialRESTPayload) GetMethodWeighting() (uint64, error) {
	// TODO(@h5law): Implement this method
	return 0, nil
}
