package proxy

import (
	"io"
	"net/http"

	"github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

// newRelayRequest builds a RelayRequest from an http.Request.
func (sync *synchronousRPCServer) newRelayRequest(request *http.Request) (*types.RelayRequest, error) {
	requestBody, err := io.ReadAll(request.Body)
	if err != nil {
		return nil, err
	}

	sync.logger.Debug().Msg("unmarshaling relay request")

	var relayReq types.RelayRequest
	if err := relayReq.Unmarshal(requestBody); err != nil {
		sync.logger.Debug().Msg("unmarshaling relay request failed")
		return nil, err
	}

	return &relayReq, nil
}

// newRelayResponse builds a RelayResponse from a response body reader and a SessionHeader.
// It also signs the RelayResponse and assigns it to RelayResponse.Meta.SupplierSignature.
// The response body is passed directly into the RelayResponse.Payload field.
func (sync *synchronousRPCServer) newRelayResponse(
	responseBody io.ReadCloser,
	sessionHeader *sessiontypes.SessionHeader,
	supplierAddr string,
) (*types.RelayResponse, error) {
	relayResponse := &types.RelayResponse{
		Meta: types.RelayResponseMetadata{SessionHeader: sessionHeader},
	}

	responsePayload, err := io.ReadAll(responseBody)
	if err != nil {
		return nil, err
	}

	relayResponse.Payload = responsePayload

	// Sign the relay response and add the signature to the relay response metadata
	if err := sync.relayerProxy.SignRelayResponse(relayResponse, supplierAddr); err != nil {
		return nil, err
	}

	return relayResponse, nil
}
