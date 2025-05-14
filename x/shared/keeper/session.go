package keeper

import (
	"context"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// GetSessionStartHeight returns the block height at which the session containing
// queryHeight starts, given the current shared onchain parameters.
// Returns 0 if the block height is not a consensus produced block.
// Example: If NumBlocksPerSession == 4, sessions start at blocks 1, 5, 9, etc.
func (k Keeper) GetSessionStartHeight(ctx context.Context, queryHeight int64) int64 {
	sharedParamsUpdates := k.GetParamsUpdates(ctx)
	return sharedtypes.GetSessionStartHeight(sharedParamsUpdates, queryHeight)
}

// GetSessionEndHeight returns the block height at which the session containing
// queryHeight ends, given the current shared onchain parameters.
// Returns 0 if the block height is not a consensus produced block.
// Example: If NumBlocksPerSession == 4, sessions end at blocks 4, 8, 11, etc.
func (k Keeper) GetSessionEndHeight(ctx context.Context, queryHeight int64) int64 {
	sharedParamsUpdates := k.GetParamsUpdates(ctx)
	return sharedtypes.GetSessionEndHeight(sharedParamsUpdates, queryHeight)
}

// GetSessionNumber returns the session number for the session containing queryHeight,
// given the current shared onchain parameters.
// Returns session number 0 if the block height is not a consensus produced block.
// Returns session number 1 for block 1 to block NumBlocksPerSession - 1 (inclusive).
// i.e. If NubBlocksPerSession == 4, session == 1 for [1, 4], session == 2 for [5, 8], etc.
func (k Keeper) GetSessionNumber(ctx context.Context, queryHeight int64) int64 {
	sharedParamsUpdates := k.GetParamsUpdates(ctx)
	return sharedtypes.GetSessionNumber(sharedParamsUpdates, queryHeight)
}

// GetProofWindowCloseHeight returns the block height at which the proof window of
// the session that includes queryHeight closes, given the passed sharedParams.
func (k Keeper) GetProofWindowCloseHeight(ctx context.Context, queryHeight int64) int64 {
	sharedParamsUpdates := k.GetParamsUpdates(ctx)
	return sharedtypes.GetProofWindowCloseHeight(sharedParamsUpdates, queryHeight)
}
