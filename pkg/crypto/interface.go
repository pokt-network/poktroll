//go:generate go run go.uber.org/mock/mockgen -destination=../../testutil/mockcrypto/ring_cache_mock.go -package=mockcrypto . RingCache
package crypto

import (
	"context"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	ring "github.com/pokt-network/ring-go"

	"github.com/pokt-network/poktroll/x/service/types"
)

// RingCache is used to store rings used for signing and verifying relay requests.
// It will cache rings for future use after querying the application module for
// the addresses of the gateways the application is delegated to, and converting
// them into their corresponding public key points on the secp256k1 curve.
type RingCache interface {
	RingClient

	// GetCachedAddresses returns the addresses of the applications that are
	// currently cached in the ring cache.
	GetCachedAddresses() []string
	// Start starts the ring cache, it takes a cancellable context and, in a
	// separate goroutine, listens for onchain delegation events and invalidates
	// the cache if the redelegation event's AppAddress is stored in the cache.
	Start(ctx context.Context)
	// Stop stops the ring cache by unsubscribing from onchain delegation events.
	// And clears the cache, so that it no longer contains any rings,
	Stop()
}

// RingClient is used to construct rings by querying the application module for
// the addresses of the gateways the application delegated to, and converting
// them into their corresponding public key points on the secp256k1 curve.
type RingClient interface {
	// GetRingForAddressAtHeight returns the ring for the given application address
	// and blockHeight if it exists.
	GetRingForAddressAtHeight(
		ctx context.Context,
		appAddress string,
		blockHeight int64,
	) (*ring.Ring, error)

	// VerifyRelayRequestSignature verifies the relay request signature against
	// the ring for the application address in the relay request.
	VerifyRelayRequestSignature(ctx context.Context, relayRequest *types.RelayRequest) error
}

// PubKeyClient is used to get the public key given an address.
// Onchain and offchain implementations should take care of retrieving the
// address' account and returning its public key.
type PubKeyClient interface {
	// GetPubKeyFromAddress returns the public key of the given account address
	// if it exists.
	GetPubKeyFromAddress(ctx context.Context, address string) (cryptotypes.PubKey, error)
}
