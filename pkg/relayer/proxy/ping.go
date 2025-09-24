package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/pokt-network/poktroll/pkg/polylog"
)

// Ping tries to dial the suppliers backend URLs to test the connection.
func (server *relayMinerHTTPServer) Ping(ctx context.Context) error {
	for _, supplierCfg := range server.serverConfig.SupplierConfigsMap {
		// Initialize the backend URLs to test with the default service config.
		backendUrls := []*url.URL{
			supplierCfg.ServiceConfig.BackendUrl,
		}

		// Add the RPC type specific service configs to the backend URLs to test, if any.
		for _, rpcTypeBackendURL := range supplierCfg.RPCTypeServiceConfigs {
			backendUrls = append(backendUrls, rpcTypeBackendURL.BackendUrl)
		}

		// Test the connectivity of all the backend URLs for the supplier.
		for _, backendUrl := range backendUrls {
			if err := server.pingBackendURL(backendUrl, supplierCfg.ServiceId); err != nil {
				return err
			}
		}

	}

	return nil
}

// pingBackendURL tests the connectivity of a backend URL for a given service ID.
func (server *relayMinerHTTPServer) pingBackendURL(backendUrl *url.URL, serviceId string) error {
	// Default client timeout for pinging the backend URL.
	const httpPingTimeout = 2 * time.Second

	// Initialize the HTTP client for the backend URL.
	c := &http.Client{Timeout: httpPingTimeout}

	// Normalize the backend URL scheme for pinging.
	// This is done to ensure that the backend URL is uses HTTP/HTTPS
	// for the ping request.
	// For example, if the backend URL is using "ws" or "wss", it will be
	// normalized to "http" or "https" respectively.
	//
	// TODO_IMPROVE: Consider testing websocket connectivity by establishing
	// a websocket connection instead of using an HTTP connection.
	pingURL := normalizeBackendURLSchemeForPing(server.logger, backendUrl)

	resp, err := c.Head(pingURL.String())
	if err != nil {
		return fmt.Errorf(
			"❌ Error pinging backend %q for serviceId %q: %w",
			backendUrl.String(), serviceId, err,
		)
	}
	_ = resp.Body.Close()

	if resp.StatusCode >= http.StatusInternalServerError {
		return fmt.Errorf(
			"❌ Error pinging backend %q for serviceId %q: received status code %d",
			backendUrl.String(), serviceId, resp.StatusCode,
		)
	}

	return nil
}

// normalizeBackendURLSchemeForPing normalizes the backend URL scheme for pinging.
// Returns a copy of the URL with the scheme normalized for HTTP connectivity checks.
// eg. "ws" -> "http", "wss" -> "https"
func normalizeBackendURLSchemeForPing(logger polylog.Logger, backendUrl *url.URL) *url.URL {
	// Create a copy of the URL to avoid modifying the original
	pingURL := *backendUrl

	if backendUrl.Scheme == "ws" || backendUrl.Scheme == "wss" {
		logger.Info().Msgf(
			"💡 backend URL %s scheme is a %s, switching to http to check connectivity",
			backendUrl.String(),
			backendUrl.Scheme,
		)

		if backendUrl.Scheme == "ws" {
			pingURL.Scheme = "http"
		} else {
			pingURL.Scheme = "https"
		}
	}

	return &pingURL
}
