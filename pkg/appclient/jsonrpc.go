package appclient

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"github.com/cometbft/cometbft/crypto"

	"pocket/x/service/types"
	sessiontypes "pocket/x/session/types"
	sharedtypes "pocket/x/shared/types"
)

// handleJSONRPCRelays handles JSON RPC relay requests.
func (app *appClient) handleJSONRPCRelays(
	ctx context.Context,
	serviceId string,
	request *http.Request,
) (responseBz []byte, err error) {
	// Read the request body bytes.
	payloadBz, err := io.ReadAll(request.Body)
	if err != nil {
		return nil, err
	}

	// Hash and sign the request payload.
	hash := crypto.Sha256(payloadBz)
	signature, _, err := app.keyring.Sign(app.keyName, hash)
	if err != nil {
		return nil, err
	}

	// Create the relay request payload.
	relayRequestPayload := &types.RelayRequest_JsonRpcPayload{}
	relayRequestPayload.JsonRpcPayload.Unmarshal(payloadBz)

	// Get the current block height to query for the current session.
	currentBlock := app.blockClient.LatestBlock(ctx)

	// Query for the current session.
	sessionQueryReq := sessiontypes.QueryGetSessionRequest{
		ApplicationAddress: app.appAddress,
		ServiceId:          &sharedtypes.ServiceId{Id: serviceId},
		BlockHeight:        currentBlock.Height(),
	}
	sessionQueryRes, err := app.sessionQuerier.GetSession(ctx, &sessionQueryReq)
	if err != nil {
		return nil, err
	}

	session := sessionQueryRes.Session

	// Get a supplier URL and address for the given service and session.
	supplierUrl, supplierAddress, err := app.getRelayerUrl(ctx, serviceId, session)
	if err != nil {
		return nil, err
	}

	// Create the relay request.
	relayRequest := &types.RelayRequest{
		Meta: &types.RelayRequestMetadata{
			SessionHeader: session.Header,
			Signature:     signature,
		},
		Payload: relayRequestPayload,
	}

	// Marshal the relay request to bytes and create a reader to be used as an HTTP request body.
	relayRequestBz, err := relayRequest.Marshal()
	if err != nil {
		return nil, err
	}
	relayRequestReader := io.NopCloser(bytes.NewReader(relayRequestBz))

	// Create the HTTP request to send the request to the relayer.
	relayHTTPRequest := &http.Request{
		Method: request.Method,
		Header: request.Header,
		URL:    supplierUrl,
		Body:   relayRequestReader,
	}

	// Perform the HTTP request to the relayer.
	relayHTTPResponse, err := http.DefaultClient.Do(relayHTTPRequest)
	if err != nil {
		return nil, err
	}

	// Read the response body bytes.
	relayResponseBz, err := io.ReadAll(relayHTTPResponse.Body)
	if err != nil {
		return nil, err
	}

	// Unmarshal the response bytes into a RelayResponse.
	relayResponse := &types.RelayResponse{}
	if err := relayResponse.Unmarshal(relayResponseBz); err != nil {
		return nil, err
	}

	// Verify the response signature. We use the supplier address that we got from
	// the getRelayerUrl function since this is the address we are expecting to sign the response.
	// TODO_TECHDEBT: if the RelayResponse is an internal error response, we should not verify the signature
	// as in some relayer early failures, it may not be signed by the supplier.
	if err := app.verifyResponse(ctx, supplierAddress, relayResponse); err != nil {
		return nil, err
	}

	// Marshal the response payload to bytes to be sent back to the application.
	var responsePayloadBz []byte
	_, err = relayResponse.Payload.MarshalTo(responsePayloadBz)

	return responsePayloadBz, err
}
