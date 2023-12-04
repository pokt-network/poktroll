package sdk

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"

	cryptocomet "github.com/cometbft/cometbft/crypto"
	"github.com/pokt-network/poktroll/pkg/signer"
	"github.com/pokt-network/poktroll/x/service/types"
)

// SendRelay sends a relay request to the given supplier's endpoint.
// It signs the request, relays it to the supplier and verifies the response signature.
// It takes an http.Request as an argument and uses its method and headers to create
// the relay request.
func (sdk *poktrollSDK) SendRelay(
	ctx context.Context,
	supplierEndpoint *SupplierEndpoint,
	request *http.Request,
) (response *types.RelayResponse, err error) {
	payloadBz, err := io.ReadAll(request.Body)
	if err != nil {
		return nil, ErrSDKHandleRelay.Wrapf("reading request body: %s", err)
	}

	// Create the relay request.
	relayRequest := &types.RelayRequest{
		Meta: &types.RelayRequestMetadata{
			SessionHeader: supplierEndpoint.Header,
			Signature:     nil, // signature added below
		},
		Payload: payloadBz,
	}

	// Get the application's signer.
	appAddress := supplierEndpoint.Header.ApplicationAddress
	appRing, err := sdk.ringCache.GetRingForAddress(ctx, appAddress)
	if err != nil {
		return nil, ErrSDKHandleRelay.Wrapf("getting app ring: %s", err)
	}
	signer := signer.NewRingSigner(appRing, sdk.signingKey)

	// Hash and sign the request's signable bytes.
	signableBz, err := relayRequest.GetSignableBytes()
	if err != nil {
		return nil, ErrSDKHandleRelay.Wrapf("getting signable bytes: %s", err)
	}

	hash := cryptocomet.Sha256(signableBz)
	signature, err := signer.Sign(hash)
	if err != nil {
		return nil, ErrSDKHandleRelay.Wrapf("signing relay: %s", err)
	}
	relayRequest.Meta.Signature = signature

	// Marshal the relay request to bytes and create a reader to be used as an HTTP request body.
	cdc := types.ModuleCdc
	relayRequestBz, err := cdc.Marshal(relayRequest)
	if err != nil {
		return nil, ErrSDKHandleRelay.Wrapf("marshaling relay request: %s", err)
	}
	relayRequestReader := io.NopCloser(bytes.NewReader(relayRequestBz))
	var relayReq types.RelayRequest
	if err := relayReq.Unmarshal(relayRequestBz); err != nil {
		return nil, ErrSDKHandleRelay.Wrapf("unmarshaling relay response: %s", err)
	}

	// Create the HTTP request to send the request to the relayer.
	relayHTTPRequest := &http.Request{
		Method: request.Method,
		Header: request.Header,
		URL:    supplierEndpoint.Url,
		Body:   relayRequestReader,
	}

	log.Printf("DEBUG: Sending signed relay request to %s", supplierEndpoint.Url)
	relayHTTPResponse, err := http.DefaultClient.Do(relayHTTPRequest)
	if err != nil {
		return nil, ErrSDKHandleRelay.Wrapf("sending relay request: %s", err)
	}

	// Read the response body bytes.
	relayResponseBz, err := io.ReadAll(relayHTTPResponse.Body)
	if err != nil {
		return nil, ErrSDKHandleRelay.Wrapf("reading relay response body: %s", err)
	}

	// Unmarshal the response bytes into a RelayResponse.
	relayResponse := &types.RelayResponse{}
	if err := relayResponse.Unmarshal(relayResponseBz); err != nil {
		return nil, ErrSDKHandleRelay.Wrapf("unmarshaling relay response: %s", err)
	}

	// Verify the response signature. We use the supplier address that we got from
	// the getRelayerUrl function since this is the address we are expecting to sign the response.
	// TODO_TECHDEBT: if the RelayResponse is an internal error response, we should not verify the signature
	// as in some relayer early failures, it may not be signed by the supplier.
	// TODO_IMPROVE: Add more logging & telemetry so we can get visibility and signal into
	// failed responses.
	if err := sdk.verifyResponse(ctx, supplierEndpoint.SupplierAddress, relayResponse); err != nil {
		// TODO_DISCUSS: should this be its own error type and asserted against in tests?
		return nil, ErrSDKHandleRelay.Wrapf("verifying relay response signature: %s", err)
	}

	return relayResponse, nil
}
