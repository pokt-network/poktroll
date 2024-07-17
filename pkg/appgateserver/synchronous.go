package appgateserver

import (
	"context"
	"net/http"

	sdktypes "github.com/pokt-network/shannon-sdk/types"

	sharedtypes "github.com/pokt-network/poktroll/proto/types/shared"
)

// requestInfo is a struct that holds the information needed to handle a relay request.
type requestInfo struct {
	appAddress  string
	serviceId   string
	rpcType     sharedtypes.RPCType
	poktRequest *sdktypes.POKTHTTPRequest
	requestBz   []byte
}

// handleSynchronousRelay handles relay requests for synchronous protocols, where
// there is a one-to-one correspondence between the request and response.
// It does everything from preparing, signing and sending the request.
// It then blocks on the response to come back and forward it to the provided writer.
func (app *appGateServer) handleSynchronousRelay(
	ctx context.Context,
	reqInfo *requestInfo,
	writer http.ResponseWriter,
) error {
	serviceId := reqInfo.serviceId
	rpcType := reqInfo.rpcType
	poktRequest := reqInfo.poktRequest
	requestBz := reqInfo.requestBz
	appAddress := reqInfo.appAddress

	relaysTotal.
		With("service_id", serviceId, "rpc_type", rpcType.String()).
		Add(1)

	sessionSuppliers, err := app.sdk.GetSessionSupplierEndpoints(ctx, appAddress, serviceId)
	if err != nil {
		return ErrAppGateHandleRelay.Wrapf("getting current session: %s", err)
	}

	// Get a supplier URL and address for the given service and session.
	supplierEndpoint, err := app.getRelayerUrl(rpcType, *sessionSuppliers, poktRequest.Url)
	if err != nil {
		return ErrAppGateHandleRelay.Wrapf("getting supplier URL: %s", err)
	}

	relayResponse, err := app.sdk.SendRelay(ctx, appAddress, supplierEndpoint, requestBz)
	// If the relayResponse is nil, it means that err is not nil and the error
	// should be handled by the appGateServer.
	if relayResponse == nil {
		return err
	}
	// Here, neither the relayResponse nor the error are nil, so the relayResponse's
	// contains the upstream service's error response.
	if err != nil {
		return ErrAppGateUpstreamError.Wrap(string(relayResponse.Payload))
	}

	// Deserialize the RelayResponse payload to get the serviceResponse that will
	// be forwarded to the client.
	serviceResponse, err := sdktypes.DeserializeHTTPResponse(relayResponse.Payload)
	if err != nil {
		return ErrAppGateHandleRelay.Wrapf("deserializing response: %s", err)
	}

	app.logger.Debug().
		Str("relay_response_payload", string(serviceResponse.BodyBz)).
		Msg("writing relay response payload")

	// Reply to the client with the service's response status code and headers.
	// At this point the AppGateServer has not generated any internal errors, so
	// the whole response will be forwarded to the client as is, including the
	// status code and headers, be it an error or not.
	serviceResponse.CopyToHTTPHeader(writer.Header())
	writer.WriteHeader(int(serviceResponse.StatusCode))

	// Transmit the service's response body to the client.
	if _, err := writer.Write(serviceResponse.BodyBz); err != nil {
		return ErrAppGateHandleRelay.Wrapf("writing relay response payload: %s", err)
	}

	relaysSuccessTotal.
		With("service_id", serviceId, "rpc_type", rpcType.String()).
		Add(1)

	return nil
}
