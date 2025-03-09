package proxy

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"time"

	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/config"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

const (
	// supplierStakeWaitTime is the time to wait for the supplier to be staked before
	// attempting to (try again to) retrieve the supplier's onchain record.
	// This is useful for testing and development purposes, where the supplier
	// may not be staked before the relay miner starts.
	supplierStakeWaitTime = 1 * time.Second

	// supplierMaxStakeWaitTimeMinutes is the time to wait before a panic is thrown
	// if the supplier is still not staked when the time elapses.
	//
	// This is intentionally a larger number because if a RelayMiner is provisioned
	// for this long (either in testing or in prod) without an associated onchain
	// supplier being stake, we need to communicate it either to the operator or
	// to the developer.
	supplierMaxStakeWaitTimeMinutes = 20 * time.Minute
)

// BuildProvidedServices builds the advertised relay servers from the supplier's onchain advertised services.
// It populates the relayerProxy's `advertisedRelayServers` map of servers for each service, where each server
// is responsible for listening for incoming relay requests and relaying them to the supported proxied service.
func (rp *relayerProxy) BuildProvidedServices(ctx context.Context) error {
	for _, supplierOperatorAddress := range rp.relayAuthenticator.GetSupplierOperatorAddresses() {
		// TODO_MAINNET: We currently block RelayMiner from starting if at least one address
		// is not staked or staked incorrectly. As node runners will maintain many different
		// suppliers on one RelayMiner, and we expect them to stake and restake often - it might
		// not be ideal to block the process from running. However, we should show warnings/errors
		// in logs (and, potentially, metrics) that their stake is different
		// from the supplier configuration. If we don't hear feedback on that prior to launching
		// MainNet it might not be that big of a deal, though.

		// Prevent the RelayMiner from stopping by waiting until its associated supplier
		// is staked and its onchain record retrieved.
		supplier, err := rp.waitForSupplierToStake(ctx, supplierOperatorAddress)
		if err != nil {
			return err
		}

		// Check that the supplier's advertised services' endpoints are present in
		// the server config and handled by a server.
		// Iterate over the supplier's advertised services then iterate over each
		// service's endpoint
		for _, service := range supplier.Services {
			for _, endpoint := range service.Endpoints {
				endpointUrl, urlErr := url.Parse(endpoint.Url)
				if urlErr != nil {
					return urlErr
				}
				found := false
				// Iterate over the server configs and check if `endpointUrl` is present
				// in any of the server config's suppliers' service's PubliclyExposedEndpoints
				for _, serverConfig := range rp.serverConfigs {
					supplierService, ok := serverConfig.SupplierConfigsMap[service.ServiceId]
					hostname := endpointUrl.Hostname()
					if ok && slices.Contains(supplierService.PubliclyExposedEndpoints, hostname) {
						found = true
						break
					}
				}

				if !found {
					return ErrRelayerProxyServiceEndpointNotHandled.Wrapf(
						"service endpoint %s not handled by the relay miner",
						endpoint.Url,
					)
				}
			}
		}
	}

	var err error
	if rp.servers, err = rp.initializeProxyServers(); err != nil {
		return err
	}

	return nil
}

// initializeProxyServers initializes the proxy servers for each server config.
func (rp *relayerProxy) initializeProxyServers() (proxyServerMap map[string]relayer.RelayServer, err error) {
	// Build a map of serviceId -> service for the supplier's advertised services

	// Build a map of listenAddress -> RelayServer for each server defined in the config file
	servers := make(map[string]relayer.RelayServer)

	// serverConfigs is a map with ListenAddress as the key which guarantees that
	// there are no duplicate servers with the same ListenAddress.
	for _, serverConfig := range rp.serverConfigs {
		rp.logger.Info().Str("server host", serverConfig.ListenAddress).Msg("starting relay proxy server")

		// Initialize the server according to the server type defined in the config file
		switch serverConfig.ServerType {
		case config.RelayMinerServerTypeHTTP:
			logger := rp.logger.With(
				"server_type", "http",
				"server_host", serverConfig.ListenAddress,
			)

			servers[serverConfig.ListenAddress] = NewHTTPServer(
				logger,
				serverConfig,
				rp.servedRelaysPublishCh,
				rp.relayAuthenticator,
				rp.relayMeter,
				rp.blockClient,
				rp.sharedQuerier,
				rp.sessionQuerier,
			)
		default:
			return nil, ErrRelayerProxyUnsupportedTransportType
		}
	}

	return servers, nil
}

// waitForSupplierToStake waits in a loop until it gets the onchain supplier's
// information back.
// This is useful for testing and development purposes, in production the supplier
// is most likely staked before the relay miner starts.
func (rp *relayerProxy) waitForSupplierToStake(
	ctx context.Context,
	supplierOperatorAddress string,
) (supplier sharedtypes.Supplier, err error) {
	startTime := time.Now()
	for {
		// Get the supplier's onchain record
		supplier, err = rp.supplierQuerier.GetSupplier(ctx, supplierOperatorAddress)

		// If the supplier is not found, wait for the supplier to be staked.
		// This enables provisioning and deploying a RelayMiner without staking a
		// supplier onchain. For testing purposes, this is particularly useful
		// to eliminate the needed of additional communication & coordination
		// between onchain staking and offchain provisioning.
		if err != nil && suppliertypes.ErrSupplierNotFound.Is(err) {
			rp.logger.Info().Msgf(
				"Waiting %d seconds for the supplier with address %s to stake",
				supplierStakeWaitTime/time.Second,
				supplierOperatorAddress,
			)
			time.Sleep(supplierStakeWaitTime)

			// See the comment above `supplierMaxStakeWaitTimeMinutes` for why
			// and how this is used.
			timeElapsed := time.Since(startTime)
			if timeElapsed > supplierMaxStakeWaitTimeMinutes {
				panic(fmt.Sprintf("Waited too long (%d minutes) for the supplier to stake. Exiting...", supplierMaxStakeWaitTimeMinutes))
			}

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
