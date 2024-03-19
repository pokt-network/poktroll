package sdk

import (
	"fmt"
	"net/url"
)

// HostToWebsocketURL converts the provided host into a websocket URL that can
// be used to subscribe to onchain events and query the chain via a client
// context or send transactions via a tx client context.
func HostToWebsocketURL(hostUrl *url.URL) string {
	if hostUrl.Scheme == "https" {
		return fmt.Sprintf("wss://%s/websocket", hostUrl.Host)
	} else {
		return fmt.Sprintf("ws://%s/websocket", hostUrl.Host)
	}
}
