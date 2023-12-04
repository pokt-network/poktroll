package appgateserver

import (
	"context"

	"github.com/pokt-network/poktroll/pkg/sdk"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// TODO_IMPROVE: This implements a naive greedy approach that defaults to the
// first available supplier.
// Future optimizations (e.g. Quality-of-Service) can be introduced here.
// TODO(@h5law): Look into different endpoint selection depending on their suitability.
// getRelayerUrl gets the URL of the relayer for the given service.
func (app *appGateServer) getRelayerUrl(
	ctx context.Context,
	serviceId string,
	rpcType sharedtypes.RPCType,
	suppliersEndpoints []*sdk.SupplierEndpoint,
) (supplierEndpoint *sdk.SupplierEndpoint, err error) {
	for _, supplierEndpoint := range suppliersEndpoints {
		// Skip services that don't match the requested serviceId.
		if supplierEndpoint.Header.Service.Id != serviceId {
			continue
		}

		// Return the first endpoint url that matches the request's RpcType.
		if supplierEndpoint.RpcType == rpcType {
			return supplierEndpoint, nil
		}
	}

	// Return an error if no relayer endpoints were found.
	return nil, ErrAppGateNoRelayEndpoints
}
