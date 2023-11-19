package partials

import (
	"github.com/pokt-network/poktroll/pkg/partials/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var (
	_ PartialPayload = (*types.PartialJSONPayload)(nil)
	_ PartialPayload = (*types.PartialRESTPayload)(nil)
)

// PartialPayload is an interface that is implemented by each of the partial
// payload types that allows for error messages to be created using the provided
// error and request payload, that matches the correct format required by the
// request type. As well as for accounting the weight of the request payload,
// which is determined by the request's method field.
type PartialPayload interface {
	GetRPCType() sharedtypes.RPCType
	GenerateErrorPayload(err error) ([]byte, error)
	GetMethodWeighting() (uint64, error)
}
