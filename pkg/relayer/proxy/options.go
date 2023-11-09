package proxy

import (
	"github.com/pokt-network/poktroll/pkg/relayer"
)

func WithSigningKeyName(keyName string) relayer.RelayerProxyOption {
	return func(relProxy relayer.RelayerProxy) {
		relProxy.(*relayerProxy).signingKeyName = keyName
	}
}

func WithProxiedServicesEndpoints(proxiedServicesEndpoints servicesEndpointsMap) relayer.RelayerProxyOption {
	return func(relProxy relayer.RelayerProxy) {
		relProxy.(*relayerProxy).proxiedServicesEndpoints = proxiedServicesEndpoints
	}
}
