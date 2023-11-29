package appgateserver

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"

	"github.com/cometbft/cometbft/crypto"

	"github.com/pokt-network/poktroll/pkg/partials"
	"github.com/pokt-network/poktroll/pkg/signer"
	"github.com/pokt-network/poktroll/x/service/types"
)

// handleSynchronousRelay handles relay requests for synchronous protocols, where
// there is a one-to-one correspondance between the request and response.
// It does everything from preparing, signing and sending the request.
// It then blocks on the response to come back and forward it to the provided writer.
func (app *appGateServer) handleSynchronousRelay(
	ctx context.Context,
	appAddress, serviceId string,
	payloadBz []byte,
	request *http.Request,
	writer http.ResponseWriter,
) error {
	// Get the type of the request by doing a partial unmarshal of the payload
	log.Printf("DEBUG: Determining request type...")
	requestType, err := partials.GetRequestType(payloadBz)
	if err != nil {
		return ErrAppGateHandleRelay.Wrapf("getting request type: %s", err)
	}
	session, err := app.getCurrentSession(ctx, appAddress, serviceId)
	if err != nil {
		return ErrAppGateHandleRelay.Wrapf("getting current session: %s", err)
	}
	log.Printf("DEBUG: Current session ID: %s", session.SessionId)

	// Get a supplier URL and address for the given service and session.
	supplierUrl, supplierAddress, err := app.getRelayerUrl(ctx, serviceId, requestType, session)
	if err != nil {
		return ErrAppGateHandleRelay.Wrapf("getting supplier URL: %s", err)
	}

	// Create the relay request.
	relayRequest := &types.RelayRequest{
		Meta: &types.RelayRequestMetadata{
			SessionHeader: session.Header,
			Signature:     nil, // signature added below
		},
		Payload: payloadBz,
	}

	// Get the application's signer.
	appRing, err := app.ringCache.GetRingForAddress(ctx, appAddress)
	if err != nil {
		return ErrAppGateHandleRelay.Wrapf("getting app ring: %s", err)
	}
	signer := signer.NewRingSigner(appRing, app.signingInformation.SigningKey)

	// Hash and sign the request's signable bytes.
	signableBz, err := relayRequest.GetSignableBytes()
	if err != nil {
		return ErrAppGateHandleRelay.Wrapf("getting signable bytes: %s", err)
	}

	hash := crypto.Sha256(signableBz)
	signature, err := signer.Sign(hash)
	if err != nil {
		return ErrAppGateHandleRelay.Wrapf("signing relay: %s", err)
	}
	relayRequest.Meta.Signature = signature

	// Marshal the relay request to bytes and create a reader to be used as an HTTP request body.
	cdc := types.ModuleCdc
	relayRequestBz, err := cdc.Marshal(relayRequest)
	if err != nil {
		return ErrAppGateHandleRelay.Wrapf("marshaling relay request: %s", err)
	}
	relayRequestReader := io.NopCloser(bytes.NewReader(relayRequestBz))
	var relayReq types.RelayRequest
	if err := relayReq.Unmarshal(relayRequestBz); err != nil {
		return ErrAppGateHandleRelay.Wrapf("unmarshaling relay response: %s", err)
	}

	// Create the HTTP request to send the request to the relayer.
	relayHTTPRequest := &http.Request{
		Method: request.Method,
		Header: request.Header,
		URL:    supplierUrl,
		Body:   relayRequestReader,
	}

	log.Printf("DEBUG: Sending signed relay request to %s", supplierUrl)
	relayHTTPResponse, err := http.DefaultClient.Do(relayHTTPRequest)
	if err != nil {
		return ErrAppGateHandleRelay.Wrapf("sending relay request: %s", err)
	}

	// Read the response body bytes.
	relayResponseBz, err := io.ReadAll(relayHTTPResponse.Body)
	if err != nil {
		return ErrAppGateHandleRelay.Wrapf("reading relay response body: %s", err)
	}

	// Unmarshal the response bytes into a RelayResponse.
	relayResponse := &types.RelayResponse{}
	if err := relayResponse.Unmarshal(relayResponseBz); err != nil {
		return ErrAppGateHandleRelay.Wrapf("unmarshaling relay response: %s", err)
	}

	// Verify the response signature. We use the supplier address that we got from
	// the getRelayerUrl function since this is the address we are expecting to sign the response.
	// TODO_TECHDEBT: if the RelayResponse is an internal error response, we should not verify the signature
	// as in some relayer early failures, it may not be signed by the supplier.
	// TODO_IMPROVE: Add more logging & telemetry so we can get visibility and signal into
	// failed responses.
	if err := app.verifyResponse(ctx, supplierAddress, relayResponse); err != nil {
		// TODO_DISCUSS: should this be its own error type and asserted against in tests?
		return ErrAppGateHandleRelay.Wrapf("verifying relay response signature: %s", err)
	}

	// Reply with the RelayResponse payload.
	log.Printf("DEBUG: Writing relay response payload: %s", string(relayResponse.Payload))
	if _, err := writer.Write(relayResponse.Payload); err != nil {
		return ErrAppGateHandleRelay.Wrapf("writing relay response payload: %s", err)
	}

	return nil
}
