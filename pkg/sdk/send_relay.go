package sdk

import (
	"bytes"
	"context"
	"io"
	"net/http"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"

	"github.com/pokt-network/poktroll/pkg/signer"
	"github.com/pokt-network/poktroll/x/service/types"
)

func init() {
	reg := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(reg)
}

// SendRelay sends a relay request to the given supplier's endpoint.
// It signs the request, relays it to the supplier and verifies the response signature.
// The relay request is created by adding method headers to the provided http.Request.
func (sdk *poktrollSDK) SendRelay(
	ctx context.Context,
	supplierEndpoint *SingleSupplierEndpoint,
	request *http.Request,
) (response *types.RelayResponse, err error) {
	// Retrieve the request's payload.
	payloadBz, err := io.ReadAll(request.Body)
	if err != nil {
		return nil, ErrSDKHandleRelay.Wrapf("reading request body: %s", err)
	}

	// Prepare the relay request.
	relayRequest := &types.RelayRequest{
		Meta: types.RelayRequestMetadata{
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

	// Hash request's signable bytes.
	signableBz, err := relayRequest.GetSignableBytesHash()
	if err != nil {
		return nil, ErrSDKHandleRelay.Wrapf("error getting signable bytes: %s", err)
	}

	// Sign the relay request.
	requestSig, err := signer.Sign(signableBz)
	if err != nil {
		return nil, ErrSDKHandleRelay.Wrapf("error signing relay: %s", err)
	}
	relayRequest.Meta.Signature = requestSig

	// Marshal the relay request to bytes and create a reader to be used as an HTTP request body.
	relayRequestBz, err := relayRequest.Marshal()
	if err != nil {
		return nil, ErrSDKHandleRelay.Wrapf("error marshaling relay request: %s", err)
	}
	relayRequestReader := io.NopCloser(bytes.NewReader(relayRequestBz))

	// Create the HTTP request to send the request to the relayer.
	// All the RPC protocols to be supported (JSONRPC, Rest, Websockets, gRPC, etc)
	// use HTTP under the hood.
	relayHTTPRequest := &http.Request{
		Method: request.Method,
		Header: request.Header,
		URL:    supplierEndpoint.Url,
		Body:   relayRequestReader,
	}

	sdk.logger.Debug().
		Str("supplier_url", supplierEndpoint.Url.String()).
		Msg("sending relay request")

	relayHTTPResponse, err := http.DefaultClient.Do(relayHTTPRequest)
	if err != nil {
		return nil, ErrSDKHandleRelay.Wrapf("error sending relay request: %s", err)
	}

	// Read the response body bytes.
	relayResponseBz, err := io.ReadAll(relayHTTPResponse.Body)
	if err != nil {
		return nil, ErrSDKHandleRelay.Wrapf("error reading relay response body: %s", err)
	}

	// Unmarshal the response bytes into a RelayResponse and validate it.
	relayResponse := &types.RelayResponse{}
	if err := relayResponse.Unmarshal(relayResponseBz); err != nil {
		return nil, ErrSDKHandleRelay.Wrapf("error unmarshaling relay response: %s", err)
	}
	if err := relayResponse.ValidateBasic(); err != nil {
		return nil, ErrSDKHandleRelay.Wrapf("%s", err)
	}

	// relayResponse.ValidateBasic validates Meta and SessionHeader, so
	// we can safely use the session header.
	sessionHeader := relayResponse.GetMeta().SessionHeader

	// Get the supplier's public key.
	supplierPubKey, err := sdk.accountQuerier.GetPubKeyFromAddress(ctx, supplierEndpoint.SupplierAddress)
	if err != nil {
		return nil, ErrSDKHandleRelay.Wrapf("error getting supplier public key: %v", err)
	}

	sdk.logger.Debug().
		Str("supplier", supplierEndpoint.SupplierAddress).
		Str("application", sessionHeader.GetApplicationAddress()).
		Str("service", sessionHeader.GetService().GetId()).
		Int64("end_height", sessionHeader.GetSessionEndBlockHeight()).
		Msg("About to verify relay response signature.")

	// Verify the relay response's supplier signature.
	// TODO_TECHDEBT: if the RelayResponse has an internal error response, we
	// SHOULD NOT verify the signature, and return an error early.
	// TODO_IMPROVE: Increase logging & telemetry get visibility into  failed responses.
	if err := relayResponse.VerifySupplierSignature(supplierPubKey); err != nil {
		return nil, ErrSDKVerifyResponseSignature.Wrapf("%s", err)
	}

	return relayResponse, nil
}
