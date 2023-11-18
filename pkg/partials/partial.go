package partials

import (
	"encoding/json"
	"log"
)

// GetErrorReply returns an error reply for the given payload and error,
// in the correct format required by the request type.
func GetErrorReply(payloadBz []byte, err error) ([]byte, error) {
	log.Printf("DEBUG: Partially Unmarshalling request: %s", string(payloadBz))
	partialRequest, er := partiallyUnmarshalRequest(payloadBz)
    if er != nil {
        return nil, er
    }
	return partialRequest.GenerateErrorPayload(err)
}

// partiallyUnmarshalRequest unmarshals the payload into a partial request
// that contains only the fields necessary to generate an error response and
// handle accounting for the request's method.
func partiallyUnmarshalRequest(payloadBz []byte) (PartialPayload, error) {
	// First attempt to unmarshal the payload into a partial JSON-RPC request
	var jsonReq partialJSONPayload
	err := json.Unmarshal(payloadBz, &jsonReq)
	// If there was no unmarshalling error and the partial request
	// is not empty then return the partial json request
	if err == nil && jsonReq != (partialJSONPayload{}) {
		// return the partial json request
		return &jsonReq, nil
	}
	// TODO(@h5law): Handle other request types
	return nil, ErrPartialUnrecognisedRequestFormat.Wrapf("got: %s", string(payloadBz))
}
