package proxy

import (
	"github.com/pokt-network/poktroll/pkg/relayer"
)

// WithSigningKeyName sets the signing key name used by the relayer proxy to sign relay responses.
// It is used along with the keyring to get the supplier address and sign the relay responses.
func WithSigningKeyName(keyName string) relayer.RelayerProxyOption {
	return func(relProxy relayer.RelayerProxy) {
		relProxy.(*relayerProxy).signingKeyName = keyName
	}
}

// WithProxiedServicesEndpoints sets the endpoints of the proxied services.
func WithProxiedServicesEndpoints(proxiedServicesEndpoints servicesEndpointsMap) relayer.RelayerProxyOption {
	return func(relProxy relayer.RelayerProxy) {
		relProxy.(*relayerProxy).proxiedServicesEndpoints = proxiedServicesEndpoints
	}
}
