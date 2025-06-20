package proxy

import (
	"io"
	"net/http"

	"github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

const (
	probabilisticDebugInfoProb = 0.001
	maxBodySize                = 1 << 20 // 1 MB limit, adjust if needed
)

// newRelayRequest builds a RelayRequest from an http.Request.
func (sync *relayMinerHTTPServer) newRelayRequest(request *http.Request) (*types.RelayRequest, error) {
	defer closeRequestBody(sync.logger, request.Body)

	limitedReader := io.LimitReader(request.Body, maxBodySize)

	requestBody, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, ErrRelayerProxyInternalError.Wrap(err.Error())
	}

	sync.logger.ProbabilisticDebugInfo(probabilisticDebugInfoProb).Msg("About to unmarshal relay request")

	var relayReq types.RelayRequest
	if err := relayReq.Unmarshal(requestBody); err != nil {
		sync.logger.Error().Err(err).Msg("❌ Unmarshaling relay request failed")
		return nil, err
	}

	sync.logger.ProbabilisticDebugInfo(probabilisticDebugInfoProb).Msg("✅ Relay request unmarshaled successfully")
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

	sync.logger.ProbabilisticDebugInfo(probabilisticDebugInfoProb).Msg("✍️ About to sign relay response")

	// Sign the relay response and add the signature to the relay response metadata
	if err := sync.relayAuthenticator.SignRelayResponse(relayResponse, supplierOperatorAddr); err != nil {
		sync.logger.Error().Err(err).Msg("❌ Signing relay response failed")
		return nil, err
	}

	sync.logger.ProbabilisticDebugInfo(probabilisticDebugInfoProb).Msg("✅ Relay response signed successfully")
	return relayResponse, nil
}
