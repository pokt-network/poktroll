package sdk

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"cosmossdk.io/depinject"
	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/pokt-network/poktroll/app"
	"github.com/pokt-network/poktroll/pkg/signer"
	"github.com/pokt-network/poktroll/x/service/types"
)

var marshaler codec.Codec

func init() {
	if err := depinject.Inject(app.AppConfig(), &marshaler); err != nil {
		panic(err)
	}
}

// SendRelay sends a relay request to the given supplier's endpoint.
// It signs the request, relays it to the supplier and verifies the response signature.
// It takes an http.Request as an argument and uses its method and headers to create
// the relay request.
func (sdk *poktrollSDK) SendRelay(
	ctx context.Context,
	supplierEndpoint *SingleSupplierEndpoint,
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
	signableBz, err := relayRequest.GetSignableBytesHash()
	if err != nil {
		return nil, ErrSDKHandleRelay.Wrapf("error getting signable bytes: %s", err)
	}

	requestSig, err := signer.Sign(signableBz)
	if err != nil {
		return nil, ErrSDKHandleRelay.Wrapf("error signing relay: %s", err)
	}
	relayRequest.Meta.Signature = requestSig

	// Marshal the relay request to bytes and create a reader to be used as an HTTP request body.
	relayRequestBz, err := marshaler.Marshal(relayRequest)
	if err != nil {
		return nil, ErrSDKHandleRelay.Wrapf("error marshaling relay request: %s", err)
	}
	relayRequestReader := io.NopCloser(bytes.NewReader(relayRequestBz))
	var relayReq types.RelayRequest
	if err := relayReq.Unmarshal(relayRequestBz); err != nil {
		return nil, ErrSDKHandleRelay.Wrapf("error unmarshaling relay request: %s", err)
	}

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

	// Unmarshal the response bytes into a RelayResponse.
	relayResponse := &types.RelayResponse{}
	if err := relayResponse.Unmarshal(relayResponseBz); err != nil {
		return nil, ErrSDKHandleRelay.Wrapf("error unmarshaling relay response: %s", err)
	}

	// Verify the response signature. We use the supplier address that we got from
	// the getRelayerUrl function since this is the address we are expecting to sign the response.
	// TODO_TECHDEBT: if the RelayResponse is an internal error response, we should not verify the signature
	// as in some relayer early failures, it may not be signed by the supplier.
	// TODO_IMPROVE: Add more logging & telemetry so we can get visibility and signal into
	// failed responses.
	if err := sdk.verifyResponse(ctx, supplierEndpoint.SupplierAddress, relayResponse); err != nil {
		return nil, ErrSDKVerifyResponseSignature.Wrapf("%s", err)
	}

	return relayResponse, nil
}
