package partials

import (
	"encoding/json"
	"log"

	"github.com/pokt-network/poktroll/pkg/partials/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// GetRequestType returns the request type for the given payload.
func GetRequestType(payloadBz []byte) (sharedtypes.RPCType, error) {
	partialRequest, err := partiallyUnmarshalRequest(payloadBz)
	if err != nil {
		return sharedtypes.RPCType_UNKNOWN_RPC, err
	}
	return partialRequest.GetRPCType(), nil
}

// GetErrorReply returns an error reply for the given payload and error,
// in the correct format required by the request type.
func GetErrorReply(payloadBz []byte, err error) ([]byte, error) {
	partialRequest, er := partiallyUnmarshalRequest(payloadBz)
	if er != nil {
		return nil, er
	}
	return partialRequest.GenerateErrorPayload(err)
}

// GetMethodWeighting returns the weighting for the given request.
func GetMethodWeighting(payloadBz []byte) (uint64, error) {
	partialRequest, err := partiallyUnmarshalRequest(payloadBz)
	if err != nil {
		return 0, err
	}
	return partialRequest.GetMethodWeighting()
}

// partiallyUnmarshalRequest unmarshals the payload into a partial request
// that contains only the fields necessary to generate an error response and
// handle accounting for the request's method.
func partiallyUnmarshalRequest(payloadBz []byte) (PartialPayload, error) {
	log.Printf("DEBUG: Partially Unmarshalling request: %s", string(payloadBz))
	// First attempt to unmarshal the payload into a partial JSON-RPC request
	var jsonReq types.PartialJSONPayload
	err := json.Unmarshal(payloadBz, &jsonReq)
	// If there was no unmarshalling error and the partial request
	// is not empty then return the partial json request
	if err == nil && jsonReq != (types.PartialJSONPayload{}) {
		// return the partial json request
		return &jsonReq, nil
	}
	// TODO(@h5law): Handle other request types
	return nil, ErrPartialUnrecognisedRequestFormat.Wrapf("got: %s", string(payloadBz))
}
