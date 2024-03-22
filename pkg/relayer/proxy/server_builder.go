package proxy

import (
	"context"
	"net/url"
	"time"

	"golang.org/x/exp/slices"

	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/config"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// supplierStakeWaitTime is the time to wait for the supplier to be staked before
// attempting to retrieve the supplier's on-chain record.
// This is useful for testing and development purposes, where the supplier
// may not be staked before the relay miner starts.
const supplierStakeWaitTime = 5

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

	// Prevent the RelayMiner from stopping by waiting until its associated supplier
	// is staked and its on-chain record retrieved.
	supplier, err := rp.waitForSupplierToStake(ctx, supplierAddress.String())
	if err != nil {
		return err
	}

	// Check that the supplier's advertised services' endpoints are present in
	// the proxy config and handled by a proxy host
	// Iterate over the supplier's advertised services then iterate over each
	// service's endpoint
	for _, service := range supplier.Services {
		for _, endpoint := range service.Endpoints {
			endpointUrl, err := url.Parse(endpoint.Url)
			if err != nil {
				return err
			}
			found := false
			// Iterate over the proxy configs and check if `endpointUrl` is present
			// in any of the proxy config's suppliers' service's hosts
			for _, proxyConfig := range rp.proxyConfigs {
				supplierService, ok := proxyConfig.Suppliers[service.Service.Id]
				if ok && slices.Contains(supplierService.Hosts, endpointUrl.Host) {
					found = true
					break
				}
			}

			if !found {
				return ErrRelayerProxyServiceEndpointNotHandled.Wrapf(
					"service endpoint %s not handled by proxy",
					endpoint.Url,
				)
			}
		}
	}

	rp.supplierAddress = supplier.Address

	if rp.proxyServers, err = rp.initializeProxyServers(supplier.Services); err != nil {
		return err
	}

	return nil
}

// initializeProxyServers initializes the proxy servers for each proxy config.
func (rp *relayerProxy) initializeProxyServers(
	supplierServices []*sharedtypes.SupplierServiceConfig,
) (proxyServerMap map[string]relayer.RelayServer, err error) {
	// Build a map of serviceId -> service for the supplier's advertised services
	supplierServiceMap := make(map[string]*sharedtypes.Service)
	for _, service := range supplierServices {
		supplierServiceMap[service.Service.Id] = service.Service
	}

	// Build a map of proxyName -> RelayServer for each proxy defined in the config file
	proxyServers := make(map[string]relayer.RelayServer)

	for _, proxyConfig := range rp.proxyConfigs {
		rp.logger.Info().Str("proxy host", proxyConfig.Host).Msg("starting relay proxy server")

		// TODO(@h5law): Implement a switch that handles all synchronous
		// RPC types in one server type and asynchronous RPC types in another
		// to create the appropriate RelayServer.
		// Initialize the proxy server according to the proxy type defined in the config file
		switch proxyConfig.Type {
		case config.ProxyTypeHTTP:
			proxyServers[proxyConfig.ProxyName] = NewSynchronousServer(
				rp.logger,
				proxyConfig,
				supplierServiceMap,
				rp.servedRelaysPublishCh,
				rp,
			)
		default:
			return nil, ErrRelayerProxyUnsupportedTransportType
		}
	}

	return proxyServers, nil
}

// waitForSupplierToStake waits in a loop until it gets the on-chain supplier's
// information back.
// This is useful for testing and development purposes, in production the supplier
// is most likely staked before the relay miner starts.
func (rp *relayerProxy) waitForSupplierToStake(
	ctx context.Context,
	supplierAddress string,
) (supplier sharedtypes.Supplier, err error) {
	for {
		// Get the supplier's on-chain record
		supplier, err = rp.supplierQuerier.GetSupplier(ctx, supplierAddress)

		// If the supplier is not found, wait for the supplier to be staked.
		if err != nil && suppliertypes.ErrSupplierNotFound.Is(err) {
			rp.logger.Info().Msgf(
				"Waiting %d seconds for the supplier with address %s to stake",
				supplierStakeWaitTime,
				supplierAddress,
			)
			time.Sleep(supplierStakeWaitTime * time.Second)
			continue
		}

		// If there is an error other than the supplier not being found, return the error
		if err != nil {
			return sharedtypes.Supplier{}, err
		}

		// If the supplier is found, break out of the wait loop.
		break
	}

	return supplier, nil
}
