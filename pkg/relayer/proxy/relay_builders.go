package proxy

import (
	"encoding/base64"
	"net/http"

	"github.com/pokt-network/poktroll/pkg/client/block"
	"github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

// newRelayRequest builds a RelayRequest from an http.Request.
func (sync *relayMinerHTTPServer) newRelayRequest(request *http.Request) (*types.RelayRequest, error) {
	// Replace DefaultMaxBodySize with config options
	requestBody, resetReadBodyPoolBytes, err := SafeRequestReadBody(sync.logger, request, sync.serverConfig.MaxBodySize)
	if err != nil {
		if resetReadBodyPoolBytes != nil {
			// Ensure buffer is returned to pool on error
			resetReadBodyPoolBytes()
		}
		return &types.RelayRequest{}, err
	}
	// Handle cleanup after SafeRequestReadBody succeeded:
	// - We must call the cleanup function to return the buffer to the pool
	// - If there was an error above, the cleanup would have already been performed internally
	// - This defer ensures proper resource management in the success case
	// - This MUST be deferred so we finish (un)marshalling before releasing the buffer
	defer resetReadBodyPoolBytes()

	sync.logger.Debug().Msg("unmarshaling relay request")

	var relayReq types.RelayRequest
	if err := relayReq.Unmarshal(requestBody); err != nil {
		bodyBzBase64 := base64.StdEncoding.EncodeToString(requestBody)
		// TODO_TECHDEBT(@red-0ne): Remove this debug log once the issue is resolved.
		sync.logger.With("body_bytes", bodyBzBase64).Debug().Msgf("unmarshaling relay request failed")
		return &types.RelayRequest{}, ErrRelayerProxyUnmarshalingRelayRequest.Wrapf(
			"failed to unmarshal relay request with body %q: %s", bodyBzBase64, err.Error(),
		)
	}

	return &relayReq, nil
}

// newRelayResponse:
// - Builds a RelayResponse from the serialized response and SessionHeader.
// - Signs the RelayResponse and assigns the signature to RelayResponse.Meta.SupplierOperatorSignature.
// - Embeds the entire serialized response (status code, headers, and body) into the RelayResponse.
func (sync *relayMinerHTTPServer) newRelayResponse(
	responseBz []byte,
	sessionHeader *sessiontypes.SessionHeader,
	supplierOperatorAddr string,
) (*types.RelayResponse, error) {
	relayResponse := &types.RelayResponse{
		Meta:    types.RelayResponseMetadata{SessionHeader: sessionHeader},
		Payload: responseBz,
	}

	chainVersion := sync.blockClient.GetChainVersion()
	if block.IsChainAfterAddPayloadHashInRelayResponse(chainVersion) {
		// Compute hash of the response payload for proof verification.
		// This hash will be stored in the RelayResponse and used during proof validation
		// to verify the integrity of the response without requiring the full payload.
		if err := relayResponse.UpdatePayloadHash(); err != nil {
			return nil, err
		}
	}

	// Sign the relay response and add the signature to the relay response metadata
	if err := sync.relayAuthenticator.SignRelayResponse(relayResponse, supplierOperatorAddr); err != nil {
		return nil, err
	}

	return relayResponse, nil
}
