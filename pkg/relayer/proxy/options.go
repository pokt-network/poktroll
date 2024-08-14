package proxy

import (
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/pkg/relayer/config"
)

// WithSigningKeyName sets the signing key name used by the relayer proxy to sign relay responses.
// It is used along with the keyring to get the supplier operator address and sign the relay responses.
func WithSigningKeyNames(keyNames []string) relayer.RelayerProxyOption {
	return func(relProxy relayer.RelayerProxy) {
		relProxy.(*relayerProxy).signingKeyNames = keyNames
	}
}

// WithServicesConfigMap updates the configurations of all the services
// the RelayMiner proxies requests to.
// servicesConfigMap is a map of server endpoints to their respective
// parsed configurations.
func WithServicesConfigMap(servicesConfigMap map[string]*config.RelayMinerServerConfig) relayer.RelayerProxyOption {
	return func(relProxy relayer.RelayerProxy) {
		relProxy.(*relayerProxy).serverConfigs = servicesConfigMap
	}
}
