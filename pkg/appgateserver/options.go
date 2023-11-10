package appgateserver

import (
	"net/url"
)

// WithSigningInformation sets the signing key and app address for the server.
func WithSigningInformation(signingInfo *SigningInformation) appGateServerOption {
	return func(appGateServer *appGateServer) {
		appGateServer.signingInformation = signingInfo
	}
}

// WithListeningUrl sets the listening URL for the app gate server.
func WithListeningUrl(listeningUrl *url.URL) appGateServerOption {
	return func(appGateServer *appGateServer) {
		appGateServer.listeningEndpoint = listeningUrl
	}
}
