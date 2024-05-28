package keeper

import (
	"context"

	"github.com/pokt-network/poktroll/x/shared"
)

// GetSessionStartHeight returns the block height at which the session starts given
// the passed shared on-chain parameters.
// Returns 0 if the block height is not a consensus produced block.
// Example: If NumBlocksPerSession == 4, sessions start at blocks 1, 5, 9, etc.
func (k Keeper) GetSessionStartHeight(ctx context.Context, queryHeight int64) int64 {
	sharedParams := k.GetParams(ctx)
	return shared.GetSessionStartHeight(&sharedParams, queryHeight)
}

// GetSessionEndHeight returns the block height at which the session ends
// given the passed shared on-chain parameters.
// Returns 0 if the block height is not a consensus produced block.
// Example: If NumBlocksPerSession == 4, sessions end at blocks 4, 8, 11, etc.
func (k Keeper) GetSessionEndHeight(ctx context.Context, queryHeight int64) int64 {
	sharedParams := k.GetParams(ctx)
	return shared.GetSessionEndHeight(&sharedParams, queryHeight)
}

// GetSessionNumber returns the session number given the block height given the passed
// shared on-chain parameters.
// Returns session number 0 if the block height is not a consensus produced block.
// Returns session number 1 for block 1 to block NumBlocksPerSession - 1 (inclusive).
// i.e. If NubBlocksPerSession == 4, session == 1 for [1, 4], session == 2 for [5, 8], etc.
func (k Keeper) GetSessionNumber(ctx context.Context, queryHeight int64) int64 {
	sharedParams := k.GetParams(ctx)
	return shared.GetSessionNumber(&sharedParams, queryHeight)
}
