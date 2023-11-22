package payloads

import (
	"encoding/json"
	"errors"
	"log"

	"github.com/pokt-network/poktroll/x/shared/types"
)

// PartialJSONPayload is a partial representation of a JSON-RPC request payload
// that contains the minimal fields necessary to generate an error response and
// handle accounting for the request's method field.
type PartialJSONPayload struct {
	Id      uint64 `json:"id"`
	JsonRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
}

// ValidateBasic ensures that all the required fields are set in the partial
// JSON payload.
// It uses a non-pointer receiver to ensure the default values of unset fields
// are present
func (j PartialJSONPayload) ValidateBasic() error {
	var err error
	if j.Id == 0 {
		err = errors.Join(err, errors.New("id field is zero"))
	}
	if j.JsonRPC == "" {
		err = errors.Join(err, errors.New("jsonrpc version field is empty"))
	}
	if j.Method == "" {
		err = errors.Join(err, errors.New("method field is empty"))
	}
	log.Printf("DEBUG: Validating basic JSON payload: %v", err)
	return err
}

// PartiallyUnmarshalJSONPayload receives a serialised payload and attempts to
// unmarshal it into the PartialJSONPayload struct. If successful this struct
// is returned, if however the struct does not contain all the required fields
// an error is returned detailing what was missing.
// If the payload is not a JSON request this function will return nil, nil
func PartiallyUnmarshalJSONPayload(payloadBz []byte) (*PartialJSONPayload, error) {
	var jsonPayload PartialJSONPayload
	err := json.Unmarshal(payloadBz, &jsonPayload)
	// Check if we can unmarshal if we cannot return nil, nil
	if err != nil || jsonPayload == (PartialJSONPayload{}) {
		return nil, nil
	}
	// return the partial json request
	return &jsonPayload, nil
}

// GetRPCType returns the request type for the given payload.
func (j *PartialJSONPayload) GetRPCType() types.RPCType {
	return types.RPCType_JSON_RPC
}

// GenerateErrorPayload creates a JSON-RPC error payload from the provided
// error with the macthing json-rpc and id fields from the request payload.
func (j *PartialJSONPayload) GenerateErrorPayload(err error) ([]byte, error) {
	reply := map[string]any{
		"jsonrpc": j.JsonRPC,
		"id":      j.Id,
		"error": map[string]any{
			"code":    -32000,
			"message": err.Error(),
			"data":    nil,
		},
	}
	return json.Marshal(reply)
}

// GetRPCComputeUnits returns the compute units for the RPC request
func (j *PartialJSONPayload) GetRPCComputeUnits() (uint64, error) {
	// TODO(@h5law): Implement this method
	return 0, nil
}
