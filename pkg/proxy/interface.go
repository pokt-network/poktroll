package proxy

import (
	"context"

	"pocket/pkg/observable"
	"pocket/x/service/types"
)

// RelayerProxy is the interface for the proxy that serves relays to the application.
// It is responsible for starting and stopping all the supported the proxies.
// Wile handling requests and responding to them in a closed loop, it also notifies
// the miner about the relays that has been served.
type RelayerProxy interface {

	// Start starts all the supported proxies or returns an error if any of them fails to start.
	Start(ctx context.Context) error

	// Stop stops all the supported proxies or returns an error if any of them fails.
	Stop() error

	// ServedRelays returns an observable that notifies the miner about the relays that has been served.
	// A served relay is a relay that its RelayRequest's signature and session has been verified
	// and its RelayResponse has been signed and successfully sent to the client.
	ServedRelays() observable.Observable[*types.Relay]
}
