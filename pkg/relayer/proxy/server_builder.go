package proxy

import (
	"context"
	"time"

	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/config"
)

const (
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

		// Check if the supplier is staked onchain and log its configured services
		rp.logSupplierServices(ctx, supplierOperatorAddress)

		// Log all the RelayMiner's configured services for the supplier.
		rp.logRelayMinerConfiguredServices(supplierOperatorAddress)
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

// logRelayMinerConfiguredServices logs the services configured in the RelayMiner
// server configs. This is useful for debugging and understanding which services
// the RelayMiner is configured to handle.
func (rp *relayerProxy) logRelayMinerConfiguredServices(supplierOperatorAddress string) {

	availableConfigs := make(map[string]struct{})
	for _, serviceConfig := range rp.serverConfigs {
		for serviceId := range serviceConfig.SupplierConfigsMap {
			availableConfigs[serviceId] = struct{}{}
		}
	}

	availableServices := make([]string, 0, len(availableConfigs))
	for serviceId := range availableConfigs {
		availableServices = append(availableServices, serviceId)
	}
	rp.logger.Info().Msgf("relayminer_configs for supplier %s: %v", supplierOperatorAddress, availableServices)

}

// logSupplierServices logs the services configured for a supplier.
// It retrieves the supplier's onchain information and logs the services that the
// supplier is configured to provide.
func (rp *relayerProxy) logSupplierServices(ctx context.Context, supplierOperatorAddress string) {
	supplier, err := rp.supplierQuerier.GetSupplier(ctx, supplierOperatorAddress)
	if err != nil {
		rp.logger.Error().Msgf(
			"failed to get Supplier with address %q onchain information: %s",
			supplierOperatorAddress,
			err.Error(),
		)
	}

	configuredServices := make([]string, 0)
	for _, serviceConfig := range supplier.Services {
		configuredServices = append(configuredServices, serviceConfig.ServiceId)
	}
	rp.logger.Info().Msgf(
		"relayminer_configs for supplier %s: %v",
		supplierOperatorAddress,
		configuredServices,
	)
}
