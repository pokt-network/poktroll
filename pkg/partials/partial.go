package partials

import (
	"context"
	"io"

	sdktypes "github.com/pokt-network/shannon-sdk/types"

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
func GetErrorReply(
	ctx context.Context,
	requestBz []byte,
	upstreamError error,
) ([]byte, error) {
	// TODO_HACK(#221): This is a hack to extract the payload from the request
	// until partials package is refactored to handle the request directly.
	request, err := sdktypes.DeserializeHTTPRequest(requestBz)
	if err != nil {
		return nil, err
	}
	payloadBz, err := io.ReadAll(request.Body)
	if err != nil {
		return nil, err
	}

	partialRequest, err := PartiallyUnmarshalRequest(ctx, payloadBz)
	if err != nil {
		return nil, err
	}
	return partialRequest.GenerateErrorPayload(upstreamError)
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
// PartiallyUnmarshalRequest unmarshals the request into a partial request
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
