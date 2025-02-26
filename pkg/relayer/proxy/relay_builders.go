package proxy

import (
	"io"
	"net/http"

	"github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

// newRelayRequest builds a RelayRequest from an http.Request.
func (sync *relayMinerHTTPServer) newRelayRequest(request *http.Request) (*types.RelayRequest, error) {
	requestBody, err := io.ReadAll(request.Body)
	if err != nil {
		return nil, ErrRelayerProxyInternalError.Wrap(err.Error())
	}

	sync.logger.Debug().Msg("unmarshaling relay request")

	var relayReq types.RelayRequest
	if err := relayReq.Unmarshal(requestBody); err != nil {
		sync.logger.Debug().Msg("unmarshaling relay request failed")
		return nil, err
	}

	return &relayReq, nil
}

// newRelayResponse builds a RelayResponse from the serialized response and SessionHeader.
// It also signs the RelayResponse and assigns it to RelayResponse.Meta.SupplierOperatorSignature.
// The whole serialized response (i.e. status code, headers and body) is embedded
// into the RelayResponse.
func (sync *relayMinerHTTPServer) newRelayResponse(
	responseBz []byte,
	sessionHeader *sessiontypes.SessionHeader,
	supplierOperatorAddr string,
) (*types.RelayResponse, error) {
	relayResponse := &types.RelayResponse{
		Meta:    types.RelayResponseMetadata{SessionHeader: sessionHeader},
		Payload: responseBz,
	}

	// Sign the relay response and add the signature to the relay response metadata
	if err := sync.relayAuthenticator.SignRelayResponse(relayResponse, supplierOperatorAddr); err != nil {
		return nil, err
	}

	return relayResponse, nil
}
