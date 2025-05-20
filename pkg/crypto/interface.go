//go:generate go run go.uber.org/mock/mockgen -destination=../../testutil/mockcrypto/ring_client_mock.go -package=mockcrypto . RingClient
package crypto

import (
	"context"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	ring "github.com/pokt-network/ring-go"

	"github.com/pokt-network/poktroll/x/service/types"
)

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
