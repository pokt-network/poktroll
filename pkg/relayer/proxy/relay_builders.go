package proxy

import (
	"encoding/base64"
	"net/http"
	"sync"

	"github.com/pokt-network/poktroll/pkg/client/block"
	"github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

var relayRequestPool = sync.Pool{
	New: func() any { return new(types.RelayRequest) },
}

var relayResponsePool = sync.Pool{
	New: func() any { return new(types.RelayResponse) },
}

// newRelayRequest builds a RelayRequest from an http.Request.
func (sync *relayMinerHTTPServer) newRelayRequest(request *http.Request) (*types.RelayRequest, func(), error) {
	// Replace DefaultMaxBodySize with config options
	requestBody, resetReadBodyPoolBytes, err := SafeRequestReadBody(sync.logger, request, sync.serverConfig.MaxBodySize)
	if err != nil {
		if resetReadBodyPoolBytes != nil {
			// Ensure buffer is returned to pool on error
			resetReadBodyPoolBytes()
		}
		return &types.RelayRequest{}, nil, err
	}
	// Handle cleanup after SafeRequestReadBody succeeded:
	// - We must call the cleanup function to return the buffer to the pool
	// - If there was an error above, the cleanup would have already been performed internally
	// - This defer ensures proper resource management in the success case
	// - This MUST be deferred so we finish (un)marshalling before releasing the buffer
	defer resetReadBodyPoolBytes()

	sync.logger.Debug().Msg("unmarshaling relay request")

	relayReq := relayRequestPool.Get().(*types.RelayRequest)
	relayReq.Reset() // ensure clean before use

	release := func() {
		relayReq.Reset()
		relayRequestPool.Put(relayReq)
	}

	if err := relayReq.Unmarshal(requestBody); err != nil {
		release()
		// TODO: encode this for just an error/log is too expensive but I understand only happens on error condition
		bodyBzBase64 := base64.StdEncoding.EncodeToString(requestBody)
		// TODO_TECHDEBT(@red-0ne): Remove this debug log once the issue is resolved.
		sync.logger.With("body_bytes", bodyBzBase64).Debug().Msgf("unmarshaling relay request failed")
		return &types.RelayRequest{}, nil, ErrRelayerProxyUnmarshalingRelayRequest.Wrapf(
			"failed to unmarshal relay request with body %q: %s", bodyBzBase64, err.Error(),
		)
	}

	return relayReq, release, nil
}

// newRelayResponse:
// - Builds a RelayResponse from the serialized response and SessionHeader.
// - Signs the RelayResponse and assigns the signature to RelayResponse.Meta.SupplierOperatorSignature.
// - Embeds the entire serialized response (status code, headers, and body) into the RelayResponse.
func (sync *relayMinerHTTPServer) newRelayResponse(
	responseBz []byte,
	sessionHeader *sessiontypes.SessionHeader,
	supplierOperatorAddr string,
) (*types.RelayResponse, func(), error) {
	relayResponse := relayRequestPool.Get().(*types.RelayResponse)
	relayResponse.Reset()

	relayResponse.Meta = types.RelayResponseMetadata{SessionHeader: sessionHeader}
	relayResponse.Payload = responseBz

	release := func() {
		relayResponse.Reset()
		relayRequestPool.Put(relayResponse)
	}

	chainVersion := sync.blockClient.GetChainVersion()
	if block.IsChainAfterAddPayloadHashInRelayResponse(chainVersion) {
		// Compute hash of the response payload for proof verification.
		// This hash will be stored in the RelayResponse and used during proof validation
		// to verify the integrity of the response without requiring the full payload.
		if err := relayResponse.UpdatePayloadHash(); err != nil {
			release()
			return nil, nil, err
		}
	}

	// Sign the relay response and add the signature to the relay response metadata
	if err := sync.relayAuthenticator.SignRelayResponse(relayResponse, supplierOperatorAddr); err != nil {
		release()
		return nil, nil, ErrRelayerProxyInternalError.Wrapf("failed to sign relay response for supplier %s: %v", supplierOperatorAddr, err)
	}

	if err := relayResponse.ValidateBasic(); err != nil {
		release()
		return nil, nil, ErrRelayerProxyInternalError.Wrapf("relay response validation failed after signing (supplier: %s): %v", supplierOperatorAddr, err)
	}

	return relayResponse, release, nil
}

func (sync *relayMinerHTTPServer) newRelayResponseWithHash(
	responseBz []byte,
	payloadHash [32]byte,
	sessionHeader *sessiontypes.SessionHeader,
	supplierOperatorAddr string,
) (*types.RelayResponse, func(), error) {
	relayResp := relayResponsePool.Get().(*types.RelayResponse)
	// always reset before reuse
	relayResp.Reset()

	relayResp.Meta = types.RelayResponseMetadata{SessionHeader: sessionHeader}
	relayResp.Payload = responseBz

	release := func() {
		relayResp.Reset()
		relayRequestPool.Put(relayResp)
	}

	// Only set hash when protocol requires it
	chainVersion := sync.blockClient.GetChainVersion()
	if block.IsChainAfterAddPayloadHashInRelayResponse(chainVersion) {
		// Compute hash of the response payload for proof verification.
		// This hash will be stored in the RelayResponse and used during proof validation
		// to verify the integrity of the response without requiring the full payload.
		relayResp.PayloadHash = payloadHash[:] // copy []byte
	}

	// Sign
	if err := sync.relayAuthenticator.SignRelayResponse(relayResp, supplierOperatorAddr); err != nil {
		release()
		return nil, nil, ErrRelayerProxyInternalError.Wrapf("failed to sign relay response for supplier %s: %v", supplierOperatorAddr, err)
	}

	// Validate
	if err := relayResp.ValidateBasic(); err != nil {
		release()
		return nil, nil, ErrRelayerProxyInternalError.Wrapf("relay response validation failed after signing (supplier: %s): %v", supplierOperatorAddr, err)
	}

	// hand ownership to caller; they should Put() when done
	return relayResp, release, nil
}
