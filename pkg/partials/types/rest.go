package types

import (
	"github.com/pokt-network/poktroll/x/shared/types"
)

// PartialRESTPayload is a partial representation of a REST request payload
// that contains only the fields necessary to generate an error response and
// handle accounting for the request's method as determined by its headers.
type PartialRESTPayload struct {
	Headers map[string]string `json:"headers"`
}

// GetRequestType returns the request type for the given payload.
func (r *PartialRESTPayload) GetRPCType() types.RPCType {
	return types.RPCType_REST_RPC
}

// GenerateErrorPayload creates a REST error payload using the headers from the
// request payload.
func (r *PartialRESTPayload) GenerateErrorPayload(err error) ([]byte, error) {
	// TODO(@h5law): Implement this method
	return nil, nil
}

// GetMethodWeighting returns the weight of the request's method from the headers.
func (r *PartialRESTPayload) GetMethodWeighting() (uint64, error) {
	// TODO(@h5law): Implement this method
	return 0, nil
}
