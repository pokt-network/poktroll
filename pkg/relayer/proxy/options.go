package proxy

import (
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/config"
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

// WithPingEnabled configures whether ping functionality is enabled for the RelayerProxy.
// When enabled, the proxy will perform health checks and connectivity tests to
// backend service endpoints.
func WithPingEnabled(pingEnabled bool) relayer.RelayerProxyOption {
	return func(relProxy relayer.RelayerProxy) {
		relProxy.(*relayerProxy).pingEnabled = pingEnabled
	}
}

// WithConnectionPoolConfig sets the HTTP connection pool configuration for the RelayerProxy.
// This allows fine-tuning of connection pooling parameters to optimize file descriptor usage.
func WithConnectionPoolConfig(connectionPoolConfig *config.RelayMinerConnectionPool) relayer.RelayerProxyOption {
	return func(relProxy relayer.RelayerProxy) {
		relProxy.(*relayerProxy).connectionPoolConfig = connectionPoolConfig
	}
}
