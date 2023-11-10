package appgateserver

import "net/url"

// WithSigningKeyName sets the signing key name for the app gate server.
func WithSigningKeyName(signingKeyName string) appGateServerOption {
	return func(appGateServer *appGateServer) {
		appGateServer.signingKeyName = signingKeyName
	}
}

// WithListeningUrl sets the listening URL for the app gate server.
func WithListeningUrl(listeningUrl *url.URL) appGateServerOption {
	return func(appGateServer *appGateServer) {
		appGateServer.listeningEndpoint = listeningUrl
	}
}
