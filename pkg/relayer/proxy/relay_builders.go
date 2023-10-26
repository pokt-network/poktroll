package proxy

import (
	"io"
	"net/http"

	"pocket/x/service/types"
	sessiontypes "pocket/x/session/types"
)

// newRelayRequest builds a RelayRequest from an http.Request.
func (j *jsonRPCServer) newRelayRequest(request *http.Request) (*types.RelayRequest, error) {
	requestBz, err := io.ReadAll(request.Body)
	if err != nil {
		return nil, err
	}

	var relayRequest types.RelayRequest
	if err := relayRequest.Unmarshal(requestBz); err != nil {
		return nil, err
	}

	return &relayRequest, nil
}

// newRelayResponse builds a RelayResponse from an http.Response and a SessionHeader.
// It also signs the RelayResponse and assigns it to RelayResponse.Meta.SupplierSignature.
// If the response has a non-nil body, it will be parsed as a JSONRPCResponsePayload.
func (j *jsonRPCServer) newRelayResponse(
	response *http.Response,
	sessionHeader *sessiontypes.SessionHeader,
) (*types.RelayResponse, error) {
	relayResponse := &types.RelayResponse{
		Meta: &types.RelayResponseMetadata{SessionHeader: sessionHeader},
	}

	responseBz, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	jsonRPCResponse := &types.JSONRPCResponsePayload{}
	if err := jsonRPCResponse.Unmarshal(responseBz); err != nil {
		return nil, err
	}

	relayResponse.Payload = &types.RelayResponse_JsonRpcPayload{JsonRpcPayload: jsonRPCResponse}

	// Sign the relay response and add the signature to the relay response metadata
	signature, err := j.relayerProxy.SignRelayResponse(relayResponse)
	if err != nil {
		return nil, err
	}
	relayResponse.Meta.SupplierSignature = signature

	return relayResponse, nil
}
