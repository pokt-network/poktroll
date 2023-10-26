package proxy

import (
	"context"

	sharedtypes "pocket/x/shared/types"
	suppliertypes "pocket/x/supplier/types"
)

// BuildProvidedServices builds the advertised relay servers from the supplier's on-chain advertised services.
// It populates the relayerProxy's `advertisedRelayServers` map of servers for each service, where each server
// is responsible for listening for incoming relay requests and relaying them to the supported proxied service.
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

	// Build the advertised relay servers map. For each service's endpoint, create the appropriate RelayServer.
	providedServices := make(relayServersMap)
	for _, serviceConfig := range services {
		serviceId := serviceConfig.Id
		proxiedServicesEndpoints := rp.proxiedServicesEndpoints[serviceId.Id]
		serviceEndpoints := make([]RelayServer, len(serviceConfig.Endpoints))

		for _, endpoint := range serviceConfig.Endpoints {
			var server RelayServer

			// Switch to the RPC type to create the appropriate RelayServer
			switch endpoint.RpcType {
			case sharedtypes.RPCType_JSON_RPC:
				server = NewJSONRPCServer(
					serviceId,
					endpoint,
					proxiedServicesEndpoints,
					rp.servedRelaysProducer,
					rp,
				)
			default:
				return ErrUnsupportedRPCType
			}

			serviceEndpoints = append(serviceEndpoints, server)
		}

		providedServices[serviceId.Id] = serviceEndpoints
	}

	rp.advertisedRelayServers = providedServices
	rp.supplierAddress = supplierQueryResponse.Supplier.Address

	return nil
}
