package partials

import (
	"context"

	"github.com/pokt-network/poktroll/pkg/partials/payloads"
	"github.com/pokt-network/poktroll/pkg/polylog"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// GetRequestType returns the request type for the given payload.
func GetRequestType(ctx context.Context, payloadBz []byte) (sharedtypes.RPCType, error) {
	partialRequest, err := PartiallyUnmarshalRequest(ctx, payloadBz)
	if err != nil {
		return sharedtypes.RPCType_UNKNOWN_RPC, err
	}
	// if the request has missing fields return an error detailing which fields
	// are missing as they are required
	if err := partialRequest.ValidateBasic(ctx); err != nil {
		return partialRequest.GetRPCType(),
			ErrPartialInvalidPayload.Wrapf("payload: %s [%v]", string(payloadBz), err)
	}
	return partialRequest.GetRPCType(), nil
}

// GetErrorReply returns an error reply for the given payload and error,
// in the correct format required by the request type.
func GetErrorReply(ctx context.Context, payloadBz []byte, err error) ([]byte, error) {
	partialRequest, er := PartiallyUnmarshalRequest(ctx, payloadBz)
	if er != nil {
		return nil, er
	}
	return partialRequest.GenerateErrorPayload(err)
}

// GetComputeUnits returns the compute units for the RPC request provided
func GetComputeUnits(ctx context.Context, payloadBz []byte) (uint64, error) {
	partialRequest, err := PartiallyUnmarshalRequest(ctx, payloadBz)
	if err != nil {
		return 0, err
	}
	// if the request has missing fields return an error detailing
	// which fields are missing
	if err := partialRequest.ValidateBasic(ctx); err != nil {
		return 0, ErrPartialInvalidPayload.Wrapf("payload: %s [%v]", string(payloadBz), err)
	}
	return partialRequest.GetRPCComputeUnits(ctx)
}

// TODO_BLOCKER(@red-0ne): This function currently only supports JSON-RPC and must
// be extended to other request types.
// PartiallyUnmarshalRequest unmarshals the payload into a partial request
// that contains only the fields necessary to generate an error response and
// handle accounting for the request's method.
func PartiallyUnmarshalRequest(ctx context.Context, payloadBz []byte) (PartialPayload, error) {
	logger := polylog.Ctx(ctx)
	logger.Debug().
		Str("payload", string(payloadBz)).
		Msg("partially Unmarshalling request")
	// First attempt to unmarshal the payload into a partial JSON-RPC request

	jsonPayload, err := payloads.PartiallyUnmarshalJSONPayload(payloadBz)
	if err != nil {
		return nil, ErrPartialInvalidPayload.Wrapf("json payload: %s [%v]", string(payloadBz), err)
	}
	if jsonPayload != nil {
		return jsonPayload, nil
	}
	// TODO_BLOCKER(@red-0ne): Handle other request types
	return nil, ErrPartialUnrecognizedRequestFormat.Wrapf("got: %s", string(payloadBz))
}
