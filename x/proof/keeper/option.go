package keeper

import "github.com/pokt-network/poktroll/pkg/crypto"

// KeeperOption is a function that can be optionally passed to the keeper constructor
// to modify its initialization behavior.
type KeeperOption func(*Keeper)

// WithRingClient overrides the RingClient that the keeper will use with the given client.
func WithRingClient(client crypto.RingClient) KeeperOption {
	return func(keeper *Keeper) {
		keeper.ringClient = client
	}
}
