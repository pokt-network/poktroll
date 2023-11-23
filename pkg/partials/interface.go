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
// payload types. It allows for partially unmarshalling the payload of various
// request types for things like error generation, request cost accounting,
// messages types, etc...
type PartialPayload interface {
	// GetRPCType returns the request type for the given payload.
	GetRPCType() sharedtypes.RPCType
	// GenerateErrorPayload creates an error message from the provided error
	// compatible with the protocol of this RPC type.
	GenerateErrorPayload(err error) ([]byte, error)
	// GetRPCComputeUnits returns the compute units for the RPC request
	GetRPCComputeUnits() (uint64, error)
	// ValidateBasic ensures that all the required fields are set in the partial
	// payload.
	ValidateBasic() error
}
