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

// WithServedRelaysBufferSize sets the buffer size of the channel that forwards
// served, reward-eligible relays into the mining pipeline. When this buffer fills,
// relays are dropped from mining (served but unpaid), so high-throughput suppliers
// should raise it. A value <= 0 keeps the observable's default buffer size.
func WithServedRelaysBufferSize(size int) relayer.RelayerProxyOption {
	return func(relProxy relayer.RelayerProxy) {
		relProxy.(*relayerProxy).servedRelaysBufferSize = size
	}
}
