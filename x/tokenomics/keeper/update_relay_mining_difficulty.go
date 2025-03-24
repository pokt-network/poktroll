package keeper

import (
	"context"

	"github.com/pokt-network/poktroll/x/service/types"
)

// UpdateRelayMiningDifficulty updates the onchain relay mining difficulty
// based on the amount of onchain relays for each service, given a map of serviceId->numRelays.
// This is a wrapper around the service keeper's UpdateRelayMiningDifficulty method
// to allow the tokenomics EndBlocker to update the relay mining difficulty after
// all claims have settled.
func (k Keeper) UpdateRelayMiningDifficulty(
	ctx context.Context,
	relaysPerServiceMap map[string]uint64,
) (difficultyPerServiceMap map[string]types.RelayMiningDifficulty, err error) {
	return k.serviceKeeper.UpdateRelayMiningDifficulty(ctx, relaysPerServiceMap)
}
