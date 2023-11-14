package proxy

import (
	"io"
	"log"
	"net/http"

	"github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

// newRelayRequest builds a RelayRequest from an http.Request.
func (jsrv *jsonRPCServer) newRelayRequest(request *http.Request) (*types.RelayRequest, error) {
	requestBz, err := io.ReadAll(request.Body)
	if err != nil {
		return nil, err
	}

	log.Printf("DEBUG: Unmarshaling relay request...")
	var relayReq types.RelayRequest
	if err := relayReq.Unmarshal(requestBz); err != nil {
		return nil, err
	}

	return &relayReq, nil
}

// newRelayResponse builds a RelayResponse from an http.Response and a SessionHeader.
// It also signs the RelayResponse and assigns it to RelayResponse.Meta.SupplierSignature.
// If the response has a non-nil body, it will be parsed as a JSONRPCResponsePayload.
func (jsrv *jsonRPCServer) newRelayResponse(
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

	log.Printf("DEBUG: Unmarshaling relay response...")
	relayResponsePayload := &types.RelayResponse_JsonRpcPayload{}
	jsonPayload := &types.JSONRPCResponsePayload{}
	cdc := types.ModuleCdc
	if err := cdc.UnmarshalJSON(responseBz, jsonPayload); err != nil {
		return nil, err
	}
	relayResponsePayload.JsonRpcPayload = jsonPayload

	relayResponse.Payload = &types.RelayResponse_JsonRpcPayload{JsonRpcPayload: jsonPayload}

	// Sign the relay response and add the signature to the relay response metadata
	if err = jsrv.relayerProxy.SignRelayResponse(relayResponse); err != nil {
		return nil, err
	}

	return relayResponse, nil
}
