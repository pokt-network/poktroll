package proxy

import (
	"context"

	sharedtypes "pocket/x/shared/types"
	suppliertypes "pocket/x/supplier/types"
)

type RelayServersMap map[string][]RelayServer

// BuildProvidedServices builds the provided services from the supplier's on-chain advertised services.
// It populates the relayerProxy's `providedServices` map of servers for each service, where each server
// is responsible for listening for incoming relay requests and poxying them to the supported native service.
func (rp *relayerProxy) BuildProvidedServices(ctx context.Context) error {
	// Get the supplier address from the keyring
	supplierAddress, err := rp.keyring.Key(rp.keyName)
	if err != nil {
		return err
	}

	// Get the supplier's advertised information from the blockchain
	supplierQuery := &suppliertypes.QueryGetSupplierRequest{Address: supplierAddress.String()}
	supplierQueryResponse, err := rp.supplierQuerier.Supplier(ctx, supplierQuery)
	if err != nil {
		return err
	}

	services := supplierQueryResponse.Supplier.Services

	// Build the provided services map. For each service's endpoint, create the appropriate server.
	providedServices := make(RelayServersMap)
	for _, serviceConfig := range services {
		serviceId := serviceConfig.Id.Id
		serviceEndpoints := make([]RelayServer, len(serviceConfig.Endpoints))

		for _, endpoint := range serviceConfig.Endpoints {
			var server RelayServer

			// Switch to the RPC type to create the appropriate server
			switch endpoint.RpcType {
			case sharedtypes.RPCType_JSON_RPC:
				server = NewJSONRPCServer(serviceId, endpoint.Url, rp)
			default:
				return ErrUnsupportedRPCType
			}

			serviceEndpoints = append(serviceEndpoints, server)
		}

		providedServices[serviceId] = serviceEndpoints
	}

	rp.providedServices = providedServices

	return nil
}
