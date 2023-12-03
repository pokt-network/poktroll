//go:generate mockgen -destination=../../testutil/mockcrypto/ring_cache_mock.go -package=mockcrypto . RingCache
package crypto

import (
	"context"

	"github.com/noot/ring-go"
)

// RingCache is used to store rings used for signing and verifying relay requests.
// It will cache rings for future use after querying the application module for
// the addresses of the gateways the application is delegated to, and converting
// them into their corresponding public key points on the secp256k1 curve.
type RingCache interface {
	// Start starts the ring cache, it takes a cancellable context and, in a
	// separate goroutine, listens for on-chain delegation events and invalidates
	// the cache if the delegatee change's address is stored in the cache.
	Start(ctx context.Context)
	// GetRingForAddress returns the ring for the given application address. If
	// it does not exist in the cache, it will be created by querying the on-chain
	// state, otherwise it will be returned from the cache.
	GetRingForAddress(ctx context.Context, appAddress string) (*ring.Ring, error)
	// Stop stops the ring cache by unsubscribing from on-chain delegation events.
	Stop()
}
