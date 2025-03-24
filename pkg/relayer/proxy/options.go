package proxy

import (
	"github.com/pokt-network/pocket/pkg/relayer"
	"github.com/pokt-network/pocket/pkg/relayer/config"
)

// WithServicesConfigMap updates the configurations of all the services
// the RelayMiner proxies requests to.
// servicesConfigMap is a map of server endpoints to their respective
// parsed configurations.
func WithServicesConfigMap(servicesConfigMap map[string]*config.RelayMinerServerConfig) relayer.RelayerProxyOption {
	return func(relProxy relayer.RelayerProxy) {
		relProxy.(*relayerProxy).serverConfigs = servicesConfigMap
	}
}
