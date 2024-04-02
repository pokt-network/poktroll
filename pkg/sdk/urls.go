package sdk

import (
	"fmt"
	"net/url"
)

// RPCToWebsocketURL converts the provided URL into a websocket URL string that can
// be used to subscribe to onchain events and query the chain via a client
// context or send transactions via a tx client context.
func RPCToWebsocketURL(hostUrl *url.URL) string {
	switch hostUrl.Scheme {
	case "http":
		return fmt.Sprintf("ws://%s/websocket", hostUrl.Host)
	case "ws":
		return fmt.Sprintf("ws://%s/websocket", hostUrl.Host)
	default:
		return fmt.Sprintf("wss://%s/websocket", hostUrl.Host)
	}
}

// ConstructGRPCUrl constructs a gRPC url string ensuring it contains either the scheme "https" or "http"
func ConstructGRPCUrl(hostUrl *url.URL) string {
	switch hostUrl.Scheme {
	case "http":
		return fmt.Sprintf("http://%s", hostUrl.Host)
	case "tcp":
		return fmt.Sprintf("tcp://%s", hostUrl.Host)
	default:
		return fmt.Sprintf("https://%s", hostUrl.Host)
	}
}
