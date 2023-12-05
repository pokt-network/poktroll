package proxy

import (
	"context"

	"github.com/pokt-network/poktroll/pkg/relayer"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// BuildProvidedServices builds the advertised relay servers from the supplier's on-chain advertised services.
// It populates the relayerProxy's `advertisedRelayServers` map of servers for each service, where each server
// is responsible for listening for incoming relay requests and relaying them to the supported proxied service.
func (rp *relayerProxy) BuildProvidedServices(ctx context.Context) error {
	// Get the supplier address from the keyring
	supplierKey, err := rp.keyring.Key(rp.signingKeyName)
	if err != nil {
		return err
	}

	supplierAddress, err := supplierKey.GetAddress()
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
		service := serviceConfig.Service
		proxiedServicesEndpoints := rp.proxiedServicesEndpoints[service.Id]
		var serviceEndpoints []relayer.RelayServer

		for _, endpoint := range serviceConfig.Endpoints {
			// url, err := url.Parse(endpoint.Url)
			// if err != nil {
			// 	return err
			// }
			// supplierEndpointHost := url.Host

			// This will throw an error if we have more than one endpoint
			supplierEndpointHost := "0.0.0.0:8545"

			rp.logger.Info().
				Fields(map[string]any{
					"service_id":   service.Id,
					"endpoint_url": endpoint.Url,
				}).
				Msg("starting relay server")

			// Switch to the RPC type
			// TODO(@h5law): Implement a switch that handles all synchronous
			// RPC types in one server type and asynchronous RPC types in another
			// to create the appropriate RelayServer
			var server relayer.RelayServer
			switch endpoint.RpcType {
			case sharedtypes.RPCType_JSON_RPC:
				server = NewSynchronousServer(
					rp.logger,
					service,
					supplierEndpointHost,
					proxiedServicesEndpoints,
					rp.servedRelaysPublishCh,
					rp,
				)
			default:
				return ErrRelayerProxyUnsupportedRPCType
			}

			serviceEndpoints = append(serviceEndpoints, server)
		}

		providedServices[service.Id] = serviceEndpoints
	}

	rp.advertisedRelayServers = providedServices
	rp.supplierAddress = supplierQueryResponse.Supplier.Address

	return nil
}
