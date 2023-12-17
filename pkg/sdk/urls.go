package sdk

import (
	"fmt"
)

// HostToWebsocketURL converts the provided host into a websocket URL that can
// be used to subscribe to onchain events and query the chain via a client
// context or send transactions via a tx client context.
func HostToWebsocketURL(host string) string {
	return fmt.Sprintf("ws://%s/websocket", host)
}
