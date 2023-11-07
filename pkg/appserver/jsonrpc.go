package appserver

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"github.com/cometbft/cometbft/crypto"

	"github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// handleJSONRPCRelay handles JSON RPC relay requests.
func (app *appServer) handleJSONRPCRelay(
	ctx context.Context,
	serviceId string,
	request *http.Request,
	writer http.ResponseWriter,
) error {
	// Read the request body bytes.
	payloadBz, err := io.ReadAll(request.Body)
	if err != nil {
		return err
	}

	// Hash and sign the request payload.
	hash := crypto.Sha256(payloadBz)
	signature, _, err := app.keyring.Sign(app.keyName, hash)
	if err != nil {
		return err
	}

	// Create the relay request payload.
	relayRequestPayload := &types.RelayRequest_JsonRpcPayload{}
	relayRequestPayload.JsonRpcPayload.Unmarshal(payloadBz)

	session, err := app.getCurrentSession(ctx, serviceId)
	if err != nil {
		return err
	}

	// Get a supplier URL and address for the given service and session.
	supplierUrl, supplierAddress, err := app.getRelayerUrl(ctx, serviceId, sharedtypes.RPCType_JSON_RPC, session)
	if err != nil {
		return err
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
		return err
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
		return err
	}

	// Read the response body bytes.
	relayResponseBz, err := io.ReadAll(relayHTTPResponse.Body)
	if err != nil {
		return err
	}

	// Unmarshal the response bytes into a RelayResponse.
	relayResponse := &types.RelayResponse{}
	if err := relayResponse.Unmarshal(relayResponseBz); err != nil {
		return err
	}

	// Verify the response signature. We use the supplier address that we got from
	// the getRelayerUrl function since this is the address we are expecting to sign the response.
	// TODO_TECHDEBT: if the RelayResponse is an internal error response, we should not verify the signature
	// as in some relayer early failures, it may not be signed by the supplier.
	if err := app.verifyResponse(ctx, supplierAddress, relayResponse); err != nil {
		return err
	}

	// Marshal the response payload to bytes to be sent back to the application.
	var responsePayloadBz []byte
	if _, err = relayResponse.Payload.MarshalTo(responsePayloadBz); err != nil {
		return err
	}

	// Reply with the RelayResponse payload.
	if _, err := writer.Write(relayRequestBz); err != nil {
		return err
	}

	return nil
}
