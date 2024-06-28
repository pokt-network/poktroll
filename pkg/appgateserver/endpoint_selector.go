package appgateserver

import (
	"net/url"
	"strconv"

	shannonsdk "github.com/pokt-network/shannon-sdk"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// getRelayerUrl returns the next relayer endpoint to use for the given serviceId and rpcType.
// NB: This is a naive implementation of the endpoint selection strategy.
// It is intentionally kept simple for the sake of a clear example, and future
// optimizations (i.e. quality of service implementations) are left as an exercise
// to gateways.
func (app *appGateServer) getRelayerUrl(
	rpcType sharedtypes.RPCType,
	sessionEndpoints shannonsdk.FilteredSession,
	requestUrlStr string,
) (supplierEndpoint shannonsdk.Endpoint, err error) {
	endpoints, err := sessionEndpoints.AllEndpoints()
	if err != nil {
		return nil, err
	}

	// Filter out the supplier endpoints that match the requested serviceId.
	matchingRPCTypeEndpoints := []shannonsdk.Endpoint{}

	for _, supplierEndpoints := range endpoints {
		for _, supplierEndpoint := range supplierEndpoints {
			// Collect the endpoints that match the request's RpcType.
			if supplierEndpoint.Endpoint().RpcType == rpcType {
				matchingRPCTypeEndpoints = append(matchingRPCTypeEndpoints, supplierEndpoint)
			}
		}
	}

	// Return an error if no relayer endpoints were found.
	if len(matchingRPCTypeEndpoints) == 0 {
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

	requestUrl, err := url.Parse(requestUrlStr)
	if err != nil {
		return nil, err
	}

	// If a `relayCount` query parameter is provided, use it to determine the next endpoint;
	// otherwise, continue the rotation based off the last selected endpoint index.
	relayCount := requestUrl.Query().Get("relayCount")
	nextEndpointIdx := 0
	if relayCount != "" {
		relayCountNum, err := strconv.Atoi(relayCount)
		if err != nil {
			relayCountNum = 0
		}
		nextEndpointIdx = relayCountNum % len(matchingRPCTypeEndpoints)
	} else {
		app.endpointSelectionIndex = (app.endpointSelectionIndex + 1) % len(matchingRPCTypeEndpoints)
		nextEndpointIdx = app.endpointSelectionIndex
	}

	return matchingRPCTypeEndpoints[nextEndpointIdx], nil
}
