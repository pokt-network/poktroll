package payloads

import (
	"context"

	"github.com/pokt-network/poktroll/x/shared/types"
)

// PartialRESTPayload is a partial representation of a REST request payload
// that contains the minimal fields necessary for basic business logic such as
// generating an  error response, handling request account, etc...
type PartialRESTPayload struct {
	Headers map[string]string `json:"headers"`
}

// PartiallyUnmarshalRESTPayload receives a serialized payload and attempts to
// unmarshal it into the PartialRESTPayload struct. If successful this struct
// is returned, if however the struct does not contain all the required fields
// the success return value is false and a nil payload is returned.
func PartiallyUnmarshalRESTPayload(payloadBz []byte) (restPayload *PartialRESTPayload, success bool) {
	// TODO(@h5law): Implement this function
	return nil, false
}

// ValidateBasic ensures that all the required fields are set in the partial
// REST payload.
// It uses a non-pointer receiver to ensure the default values of unset fields
// are present
func (r PartialRESTPayload) ValidateBasic(ctx context.Context) error {
	// TODO(@h5law): Implement this function
	var err error
	return err
}

// GetRPCType returns the request type for the given payload.
func (r *PartialRESTPayload) GetRPCType() types.RPCType {
	return types.RPCType_REST
}

// GenerateErrorPayload creates a REST error payload using the headers from the
// request payload.
func (r *PartialRESTPayload) GenerateErrorPayload(err error) ([]byte, error) {
	// TODO(@h5law): Implement this method
	return nil, nil
}

// GetRPCComputeUnits returns the compute units for the RPC request
func (r *PartialRESTPayload) GetRPCComputeUnits(ctx context.Context) (uint64, error) {
	// TODO(@h5law): Implement this method
	return 0, nil
}
