package proxy

import (
	"context"

	"pocket/pkg/observable"
	"pocket/x/service/types"
)

// RelayerProxy is the interface for the proxy that serves relays to the application.
// It is responsible for starting and stopping all supported proxies.
// While handling requests and responding in a closed loop, it also notifies
// the miner about the relays that have been served.
type RelayerProxy interface {

	// Start starts all supported proxies and returns an error if any of them fail to start.
	Start(ctx context.Context) error

	// Stop stops all supported proxies and returns an error if any of them fail.
	Stop(ctx context.Context) error

	// ServedRelays returns an observable that notifies the miner about the relays that have been served.
	// A served relay is one whose RelayRequest's signature and session have been verified,
	// and its RelayResponse has been signed and successfully sent to the client.
	ServedRelays() observable.Observable[*types.Relay]
}
