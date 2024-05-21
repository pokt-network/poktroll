package appgateserver

import (
	"net/http"
	"strconv"

	"github.com/pokt-network/poktroll/pkg/sdk"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// TODO_IMPROVE: Use a more sophisticated endpoint selection strategy.
// Future optimizations (e.g. Quality-of-Service) can be introduced here.
// TODO(@h5law): Look into different endpoint selection depending on their suitability.
// getRelayerUrl gets the URL of the relayer for the given service.
func (app *appGateServer) getRelayerUrl(
	serviceId string,
	rpcType sharedtypes.RPCType,
	supplierEndpoints []*sdk.SingleSupplierEndpoint,
	request *http.Request,
) (supplierEndpoint *sdk.SingleSupplierEndpoint, err error) {
	// Filter out the supplier endpoints that match the requested serviceId.
	validSupplierEndpoints := make([]*sdk.SingleSupplierEndpoint, 0, len(supplierEndpoints))

	for _, supplierEndpoint := range supplierEndpoints {
		// Skip services that don't match the requested serviceId.
		if supplierEndpoint.Header.Service.Id != serviceId {
			continue
		}

		// Collect the endpoints that match the request's RpcType.
		if supplierEndpoint.RpcType == rpcType {
			validSupplierEndpoints = append(validSupplierEndpoints, supplierEndpoint)
		}
	}

	// Return an error if no relayer endpoints were found.
	if len(validSupplierEndpoints) == 0 {
		return nil, ErrAppGateNoRelayEndpoints
	}

	// Protect the endpointSelectionIndex update from concurrent relay requests.
	app.endpointSelectionIndexMu.Lock()
	defer app.endpointSelectionIndexMu.Unlock()

	// Select the next endpoint in the list by rotating the index.
	// This does not necessarily start from the first endpoint of a new session
	// but will cycle through all valid endpoints of the same session if enough
	// requests are made.
	// This is a naive strategy that is used to ensure all endpoints are leveraged
	// throughout the lifetime of the session. It is primarily used as a foundation
	// for testing or development purposes but a more enhanced strategy is expected
	// to be adopted by prod gateways.

	// If a `relayCount` query parameter is provided, use it to determine the next endpoint;
	// otherwise, continue the rotation based off the last selected endpoint index.
	relayCount := request.URL.Query().Get("relayCount")
	nextEndpointIdx := 0
	if relayCount != "" {
		relayCountNum, err := strconv.Atoi(relayCount)
		if err != nil {
			relayCountNum = 0
		}
		nextEndpointIdx = relayCountNum % len(validSupplierEndpoints)
	} else {
		app.endpointSelectionIndex = (app.endpointSelectionIndex + 1) % len(validSupplierEndpoints)
		nextEndpointIdx = app.endpointSelectionIndex
	}

	return validSupplierEndpoints[nextEndpointIdx], nil
}
