package appgateserver

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/cometbft/cometbft/crypto"

	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// InterimJSONRPCRequestPayload is a partial JSON RPC request payload that
// excludes the params field, which is unmarshaled sperately.
type InterimJSONRPCRequestPayload struct {
	ID      uint32          `json:"id"`
	Jsonrpc string          `json:"jsonrpc"`
	Params  json.RawMessage `json:"params"`
	Method  string          `json:"method"`
}

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
		return ErrAppGateHandleRelay.Wrapf("reading relay request body: %s", err)
	}
	log.Printf("DEBUG: relay request body: %s", string(payloadBz))

	// Create the relay request payload.
	relayRequestPayload := &types.RelayRequest_JsonRpcPayload{}
	jsonPayload := &types.JSONRPCRequestPayload{}

	// Unmarshal the request body bytes into an InterimJSONRPCRequestPayload.
	interimPayload := &InterimJSONRPCRequestPayload{}
	if err := json.Unmarshal(payloadBz, interimPayload); err != nil {
		return err
	}
	jsonPayload.Jsonrpc = interimPayload.Jsonrpc
	jsonPayload.Method = interimPayload.Method
	jsonPayload.Id = interimPayload.ID

	// Set the relay json payload's JSON RPC payload params.
	var mapParams map[string]string
	if err := json.Unmarshal(interimPayload.Params, &mapParams); err == nil {
		jsonPayload.Params = &types.JSONRPCRequestPayload_MapParams{
			MapParams: &types.JSONRPCRequestPayloadParamsMap{
				Params: mapParams,
			},
		}
	} else {
		// Try unmarshaling into a list
		var listParams []string
		if err := json.Unmarshal(interimPayload.Params, &listParams); err == nil {
			jsonPayload.Params = &types.JSONRPCRequestPayload_ListParams{
				ListParams: &types.JSONRPCRequestPayloadParamsList{
					Params: listParams,
				},
			}
		} else {
			// Neither a map nor a list
			return ErrAppGateHandleRelay.Wrapf("params must be either a map or a list of strings: %v", err)
		}
	}

	// Set the relay request payload's JSON RPC payload.
	relayRequestPayload.JsonRpcPayload = jsonPayload

	session, err := app.getCurrentSession(ctx, appAddress, serviceId)
	if err != nil {
		return ErrAppGateHandleRelay.Wrapf("getting current session: %s", err)
	}
	log.Printf("DEBUG: Current session ID: %s", session.SessionId)

	// Get a supplier URL and address for the given service and session.
	supplierUrl, supplierAddress, err := app.getRelayerUrl(ctx, serviceId, sharedtypes.RPCType_JSON_RPC, session)
	if err != nil {
		return ErrAppGateHandleRelay.Wrapf("getting supplier URL: %s", err)
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
		return ErrAppGateHandleRelay.Wrapf("getting signer: %s", err)
	}

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
	relayRequestBz := cdc.MustMarshalJSON(relayRequest)
	relayRequestReader := io.NopCloser(bytes.NewReader(relayRequestBz))
	var relayReq types.RelayRequest
	if err := cdc.UnmarshalJSON(relayRequestBz, &relayReq); err != nil {
		return ErrAppGateHandleRelay.Wrapf("unmarshaling relay request: %s", err)
	}
	log.Printf("DEBUG: relay request payload: %s", relayReq.GetJsonRpcPayload())

	// Create the HTTP request to send the request to the relayer.
	relayHTTPRequest := &http.Request{
		Method: request.Method,
		Header: request.Header,
		URL:    supplierUrl,
		Body:   relayRequestReader,
	}

	// Perform the HTTP request to the relayer.
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

	// Marshal the response payload to bytes to be sent back to the application.
	relayResponsePayloadBz, err := cdc.MarshalJSON(relayResponse.GetJsonRpcPayload())
	if err != nil {
		return ErrAppGateHandleRelay.Wrapf("unmarshallig relay response: %s", err)
	}

	// Reply with the RelayResponse payload.
	log.Printf("DEBUG: Writing relay response payload: %s", string(relayResponsePayloadBz))
	if _, err := writer.Write(relayResponsePayloadBz); err != nil {
		return ErrAppGateHandleRelay.Wrapf("writing relay response payload: %s", err)
	}

	return nil
}
