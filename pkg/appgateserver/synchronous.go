package appgateserver

import (
	"context"
	"log"
	"net/http"

	"github.com/pokt-network/poktroll/pkg/partials"
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

	sessionSuppliers, err := app.sdk.GetSessionSupplierEndpoints(ctx, appAddress, serviceId)
	if err != nil {
		return ErrAppGateHandleRelay.Wrapf("getting current session: %s", err)
	}

	// Get a supplier URL and address for the given service and session.
	supplierEndpoint, err := app.getRelayerUrl(ctx, serviceId, requestType, sessionSuppliers)
	if err != nil {
		return ErrAppGateHandleRelay.Wrapf("getting supplier URL: %s", err)
	}

	relayResponse, err := app.sdk.SendRelay(ctx, supplierEndpoint, request)
	if err != nil {
		return err
	}

	// Reply with the RelayResponse payload.
	log.Printf("DEBUG: Writing relay response payload: %s", string(relayResponse.Payload))
	if _, err := writer.Write(relayResponse.Payload); err != nil {
		return ErrAppGateHandleRelay.Wrapf("writing relay response payload: %s", err)
	}

	return nil
}
