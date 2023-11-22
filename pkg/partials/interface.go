package partials

import (
	"github.com/pokt-network/poktroll/pkg/partials/payloads"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var (
	_ PartialPayload = (*payloads.PartialJSONPayload)(nil)
	_ PartialPayload = (*payloads.PartialRESTPayload)(nil)
)

// PartialPayload is an interface that is implemented by each of the partial
// payload types that allows for error messages to be created using the provided
// error and request payload, that matches the correct format required by the
// request type. As well as for accounting the weight of the request payload,
// which is determined by the request's method field.
type PartialPayload interface {
	// GetRPCType returns the request type for the given payload.
	GetRPCType() sharedtypes.RPCType
	// GenerateErrorPayload creates an error message from the provided error
	// in the format of the request type.
	GenerateErrorPayload(err error) ([]byte, error)
	// GetRPCComputeUnits returns the compute units for the RPC request
	GetRPCComputeUnits() (uint64, error)
	// ValidateBasic ensures that all the required fields are set in the partial
	// payload.
	ValidateBasic() error
}
