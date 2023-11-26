package partials

import (
	"log"

	"github.com/pokt-network/poktroll/pkg/partials/payloads"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// GetRequestType returns the request type for the given payload.
func GetRequestType(payloadBz []byte) (sharedtypes.RPCType, error) {
	partialRequest, err := PartiallyUnmarshalRequest(payloadBz)
	if err != nil {
		return sharedtypes.RPCType_UNKNOWN_RPC, err
	}
	// if the request has missing fields return an error detailing which fields
	// are missing as they are required
	if err := partialRequest.ValidateBasic(); err != nil {
		return partialRequest.GetRPCType(),
			ErrPartialInvalidPayload.Wrapf("payload: %s [%v]", string(payloadBz), err)
	}
	return partialRequest.GetRPCType(), nil
}

// GetErrorReply returns an error reply for the given payload and error,
// in the correct format required by the request type.
func GetErrorReply(payloadBz []byte, err error) ([]byte, error) {
	partialRequest, er := PartiallyUnmarshalRequest(payloadBz)
	if er != nil {
		return nil, er
	}
	return partialRequest.GenerateErrorPayload(err)
}

// GetComputeUnits returns the compute units for the RPC request provided
func GetComputeUnits(payloadBz []byte) (uint64, error) {
	partialRequest, err := PartiallyUnmarshalRequest(payloadBz)
	if err != nil {
		return 0, err
	}
	// if the request has missing fields return an error detailing
	// which fields are missing
	if err := partialRequest.ValidateBasic(); err != nil {
		return 0, ErrPartialInvalidPayload.Wrapf("payload: %s [%v]", string(payloadBz), err)
	}
	return partialRequest.GetRPCComputeUnits()
}

// TODO_BLOCKER(@h5law): This function currently only supports JSON-RPC and must
// be extended to other request types.
// PartiallyUnmarshalRequest unmarshals the payload into a partial request
// that contains only the fields necessary to generate an error response and
// handle accounting for the request's method.
func PartiallyUnmarshalRequest(payloadBz []byte) (PartialPayload, error) {
	log.Printf("DEBUG: Partially Unmarshalling request: %s", string(payloadBz))
	// First attempt to unmarshal the payload into a partial JSON-RPC request
	jsonPayload, err := payloads.PartiallyUnmarshalJSONPayload(payloadBz)
	if err != nil {
		return nil, ErrPartialInvalidPayload.Wrapf("json payload: %s [%v]", string(payloadBz), err)
	}
	if jsonPayload != nil {
		return jsonPayload, nil
	}
	// TODO(@h5law): Handle other request types
	return nil, ErrPartialUnrecognisedRequestFormat.Wrapf("got: %s", string(payloadBz))
}
