package appgateserver

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"

	"github.com/cometbft/cometbft/crypto"

	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// handleJSONRPCRelay handles JSON RPC relay requests.
// It does everything from preparing, signing and sending the request.
// It then blocks on the response to come back and forward it to the provided writer.
func (app *appGateServer) handleJSONRPCRelay(
	ctx context.Context,
	appAddress, serviceId string,
	request *http.Request,
	writer http.ResponseWriter,
) error {
	// Read the request body bytes.
	payloadBz, err := io.ReadAll(request.Body)
	if err != nil {
		log.Println("ERROR: Failed reading relay request body")
		return err
	}
	log.Printf("DEBUG: relay request body: %s", string(payloadBz))

	// Create the relay request payload.
	relayRequestPayload := &types.RelayRequest_JsonRpcPayload{}
	jsonPayload := &types.JSONRPCRequestPayload{}
	cdc := types.ModuleCdc
	if err := cdc.UnmarshalJSON(payloadBz, jsonPayload); err != nil {
		return err
	}
	relayRequestPayload.JsonRpcPayload = jsonPayload

	session, err := app.getCurrentSession(ctx, appAddress, serviceId)
	if err != nil {
		log.Println("ERROR: Failed getting current session")
		return err
	}
	log.Printf("DEBUG: Current session ID: %s", session.SessionId)

	// Get a supplier URL and address for the given service and session.
	supplierUrl, supplierAddress, err := app.getRelayerUrl(ctx, serviceId, sharedtypes.RPCType_JSON_RPC, session)
	if err != nil {
		log.Println("ERROR: Failed getting relayer URL")
		return err
	}

	// Create the relay request.
	relayRequest := &types.RelayRequest{
		Meta: &types.RelayRequestMetadata{
			SessionHeader: session.Header,
			Signature:     nil, // signature added below
		},
		Payload: relayRequestPayload,
	}

	// Get the application's signer.
	signer, err := app.getRingSingerForAppAddress(ctx, appAddress)
	if err != nil {
		log.Println("ERROR: Failed getting signer")
		return err
	}

	// Hash and sign the request's signable bytes.
	log.Printf("DEBUG: Signing relay request...")
	signableBz, err := relayRequest.GetSignableBytes()
	if err != nil {
		log.Println("ERROR: Failed getting signable bytes")
		return err
	}

	hash := crypto.Sha256(signableBz)
	signature, err := signer.Sign(hash)
	if err != nil {
		log.Println("ERROR: Failed signing relay request")
		return err
	}
	relayRequest.Meta.Signature = signature

	// log.Printf("DEBUG: relayRequest: %+v", relayRequest)

	// Marshal the relay request to bytes and create a reader to be used as an HTTP request body.
	relayRequestBz, err := cdc.Marshal(relayRequest)
	if err != nil {
		log.Println("ERROR: Failed marshaling relay request")
		return err
	}
	relayRequestReader := io.NopCloser(bytes.NewReader(relayRequestBz))
	var relayReq types.RelayRequest
	if err := relayReq.Unmarshal(relayRequestBz); err != nil {
		return err
	}
	// log.Printf("DEBUG: Signed relay request: %+v", relayReq)

	// Create the HTTP request to send the request to the relayer.
	relayHTTPRequest := &http.Request{
		Method: request.Method,
		Header: request.Header,
		URL:    supplierUrl,
		Body:   relayRequestReader,
	}

	// log.Printf("DEBUG: relayHTTPRequest: %+v", relayHTTPRequest)

	// Perform the HTTP request to the relayer.
	log.Printf("DEBUG: Sending signed relay request to %s", supplierUrl)
	relayHTTPResponse, err := http.DefaultClient.Do(relayHTTPRequest)
	if err != nil {
		log.Println("ERROR: Failed sending relay request to relayer")
		return err
	}

	// Read the response body bytes.
	log.Printf("DEBUG: Reading relay response body...")
	relayResponseBz, err := io.ReadAll(relayHTTPResponse.Body)
	if err != nil {
		log.Println("ERROR: Failed reading relay response body")
		return err
	}

	// Unmarshal the response bytes into a RelayResponse.
	relayResponse := &types.RelayResponse{}
	if err := relayResponse.Unmarshal(relayResponseBz); err != nil {
		log.Println("ERROR: Failed unmarshaling relay response")
		return err
	}

	// Verify the response signature. We use the supplier address that we got from
	// the getRelayerUrl function since this is the address we are expecting to sign the response.
	// TODO_TECHDEBT: if the RelayResponse is an internal error response, we should not verify the signature
	// as in some relayer early failures, it may not be signed by the supplier.
	// TODO_IMPROVE: Add more logging & telemetry so we can get visibility and signal into
	// failed responses.
	if err := app.verifyResponse(ctx, supplierAddress, relayResponse); err != nil {
		log.Println("ERROR: Failed verifying relay response signature")
		return err
	}

	// Marshal the response payload to bytes to be sent back to the application.
	relayResponsePayloadBz, err := cdc.MarshalJSON(relayResponse.GetJsonRpcPayload())
	if err != nil {
		log.Println("ERROR: Failed unmarshaling relay response")
		return err
	}

	// Reply with the RelayResponse payload.
	log.Printf("DEBUG: Writing relay response payload: %s", string(relayResponsePayloadBz))
	if _, err := writer.Write(relayResponsePayloadBz); err != nil {
		log.Println("ERROR: Failed writing relay response payload to writer")
		return err
	}

	return nil
}
