package rings

import (
	"context"

	"github.com/noot/ring-go"
)

// RingCache is used to store rings used for signing and verifying relay requests.
// It will cache rings for future use after querying the application module for
// the addresses of the gateways the application is delegated to, and converting
// them into their corresponding public key points on the secp256k1 curve.
type RingCache interface {
	GetRingForAddress(ctx context.Context, appAddress string) (*ring.Ring, error)
}
