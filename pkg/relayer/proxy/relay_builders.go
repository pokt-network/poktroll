package proxy

import (
	"io"
	"log"
	"net/http"

	"github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

// newRelayRequest builds a RelayRequest from an http.Request.
func (sync *synchronousRPCServer) newRelayRequest(request *http.Request) (*types.RelayRequest, error) {
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
// The response's Body is passed directly into the RelayResponse.Payload field.
func (sync *synchronousRPCServer) newRelayResponse(
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

	relayResponse.Payload = responseBz

	// Sign the relay response and add the signature to the relay response metadata
	if err = sync.relayerProxy.SignRelayResponse(relayResponse); err != nil {
		return nil, err
	}

	return relayResponse, nil
}
