package appclient

import (
	"context"
	"net/url"

	sessiontypes "pocket/x/session/types"
	sharedtypes "pocket/x/shared/types"
)

// getRelayerUrl gets the URL of the relayer for the given service.
// It gets the suppliers list from the current session and returns
// the first relayer URL that matches the JSON RPC RpcType.
func (app *appClient) getRelayerUrl(
	ctx context.Context,
	serviceId string,
	session *sessiontypes.Session,
) (supplierUrl *url.URL, supplierAddress string, err error) {
	for _, supplier := range session.Suppliers {
		for _, service := range supplier.Services {
			// Skip services that don't match the requested serviceId.
			if service.ServiceId.Id != serviceId {
				continue
			}

			for _, endpoint := range service.Endpoints {
				// Return the first endpoint url that matches the JSON RPC RpcType.
				if endpoint.RpcType == sharedtypes.RPCType_JSON_RPC {
					supplierUrl, err := url.Parse(endpoint.Url)
					return supplierUrl, supplier.Address, err
				}
			}
		}
	}

	// Return an error if no relayer endpoints were found.
	return nil, "", ErrNoRelayEndpoints
}
