package appgateserver

import (
	"context"
	"net/http"
	"strings"

	httpcodec "github.com/pokt-network/shannon-sdk/httpcodec"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// handleSynchronousRelay handles relay requests for synchronous protocols, where
// there is a one-to-one correspondence between the request and response.
// It does everything from preparing, signing and sending the request.
// It then blocks on the response to come back and forward it to the provided writer.
func (app *appGateServer) handleSynchronousRelay(
	ctx context.Context,
	appAddress, serviceId string,
	rpcType sharedtypes.RPCType,
	request *http.Request,
	writer http.ResponseWriter,
) error {
	relaysTotal.
		With("service_id", serviceId, "rpc_type", rpcType.String()).
		Add(1)

	// TODO_IMPROVE: log additional info?
	app.logger.Debug().
		Str("rpc_type", rpcType.String()).
		Msg("got request type")

	sessionSuppliers, err := app.sdk.GetSessionSupplierEndpoints(ctx, appAddress, serviceId)
	if err != nil {
		return ErrAppGateHandleRelay.Wrapf("getting current session: %s", err)
	}

	// Get a supplier URL and address for the given service and session.
	endpoints := sessionSuppliers.SuppliersEndpoints
	supplierEndpoint, err := app.getRelayerUrl(serviceId, rpcType, endpoints, request)
	if err != nil {
		return ErrAppGateHandleRelay.Wrapf("getting supplier URL: %s", err)
	}

	// Serialize the request to be sent to the supplier as a RelayRequest.Payload
	// which will include the url, request body, method, and headers.
	requestBz, err := httpcodec.SerializeHTTPRequest(request)
	if err != nil {
		return ErrAppGateHandleRelay.Wrapf("serializing request: %s", err)
	}

	relayResponse, err := app.sdk.SendRelay(ctx, supplierEndpoint, requestBz)
	if err != nil {
		return err
	}

	// Deserialize the RelayResponse payload to get the serviceResponse that will
	// be forwarded to the client.
	serviceResponse, err := httpcodec.DeserializeHTTPResponse(relayResponse.Payload)
	if err != nil {
		return ErrAppGateHandleRelay.Wrapf("deserializing response: %s", err)
	}

	app.logger.Debug().
		Str("relay_response_payload", string(serviceResponse.Body)).
		Msg("writing relay response payload")

	// Reply to the client with the service's response status code and headers.
	// At this point the AppGateServer has not generated any internal errors, so
	// the whole response will be forwarded to the client as is, including the
	// status code and headers, be it an error or not.
	writer.WriteHeader(int(serviceResponse.StatusCode))
	for key, valuesStr := range serviceResponse.Header {
		values := strings.Split(valuesStr, ",")
		for _, value := range values {
			writer.Header().Add(key, value)
		}
	}

	// Transmit the service's response body to the client.
	if _, err := writer.Write(serviceResponse.Body); err != nil {
		return ErrAppGateHandleRelay.Wrapf("writing relay response payload: %s", err)
	}

	relaysSuccessTotal.
		With("service_id", serviceId, "rpc_type", rpcType.String()).
		Add(1)

	return nil
}
