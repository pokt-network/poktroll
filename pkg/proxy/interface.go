package proxy

import (
	"context"

	"pocket/pkg/observable"
	"pocket/x/supplier/types"
)

// Proxy interface specifies the methods that a proxy must implement. It does not assume
// how it is getting the relays (http/ws/grpc...), which would be part of the constructor function.
type Proxy interface {
	Start(ctx context.Context) error
	Close() error
	// ProcessedRelays returns a subscription to the relays that has been processed
	ProcessedRelays() observable.Subscription[*types.Relay]
}
