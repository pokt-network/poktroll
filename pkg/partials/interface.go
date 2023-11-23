package partials

import (
	"github.com/pokt-network/poktroll/pkg/partials/payloads"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var (
	_ PartialPayload = (*payloads.PartialJSONPayload)(nil)
	_ PartialPayload = (*payloads.PartialRESTPayload)(nil)
)

// PartialPayload defines an interface for partial RPC payloads that enables the
// transparent relaying of RPC requests from applications to suppliers. In order
// for this to occur we must be able to infer its format. This requires the RPC
// payload to be partially decoded, extracting the required fields, currently
// limited for the purpose of determine the RPC type, compute units and error
// generation, but may be used for other logic in the future
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
