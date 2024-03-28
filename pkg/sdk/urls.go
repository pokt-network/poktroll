package sdk

import (
	"fmt"
	"net/url"
)

// RPCToWebsocketURL converts the provided URL into a websocket URL string that can
// be used to subscribe to onchain events and query the chain via a client
// context or send transactions via a tx client context.
func RPCToWebsocketURL(hostUrl *url.URL) string {
	if hostUrl.Scheme == "http" {
		return fmt.Sprintf("ws://%s/websocket", hostUrl.Host)
	} else if hostUrl.Scheme == "ws" {
		return fmt.Sprintf("ws://%s/websocket", hostUrl.Host)
	}

	return fmt.Sprintf("wss://%s/websocket", hostUrl.Host)
}

// ConstructGRPCUrl constructs a gRPC url string ensuring it contains either the scheme "grpcs" or "grpc"
// This allows the SDK client control whether a TLS-enabled connection is used and some flexibility when specifying the gRPC URL.
func ConstructGRPCUrl(hostUrl *url.URL) string {
	if hostUrl.Scheme == "http" {
		return fmt.Sprintf("grpc://%s", hostUrl.Host)
	} else if hostUrl.Scheme == "grpc" {
		return fmt.Sprintf("grpc://%s", hostUrl.Host)
	} else if hostUrl.Scheme == "tcp" {
		return fmt.Sprintf("tcp://%s", hostUrl.Host)
	}

	return fmt.Sprintf("grpcs://%s", hostUrl.Host)
}
