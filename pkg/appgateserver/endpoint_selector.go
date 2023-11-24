package appgateserver

import (
	"context"
	"log"
	"net/url"

	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
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
	session *sessiontypes.Session,
) (supplierUrl *url.URL, supplierAddress string, err error) {
	for _, supplier := range session.Suppliers {
		for _, service := range supplier.Services {
			// Skip services that don't match the requested serviceId.
			if service.Service.Id != serviceId {
				continue
			}

			for _, endpoint := range service.Endpoints {
				// Return the first endpoint url that matches the request's RpcType.
				if endpoint.RpcType == rpcType {
					supplierUrl, err := url.Parse(endpoint.Url)
					if err != nil {
						log.Printf("ERROR: error parsing url: %s", err)
						continue
					}
					return supplierUrl, supplier.Address, nil
				}
			}
		}
	}

	// Return an error if no relayer endpoints were found.
	return nil, "", ErrAppGateNoRelayEndpoints
}
