package shared

import (
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// SessionGracePeriodBlocks is the number of blocks after the session ends before the
// "session grace period" is considered to have elapsed.
//
// TODO_BLOCKER: This is a place-holder that will be removed once the respective
// governance parameter is implemented.
const SessionGracePeriodBlocks = 4

// GetSessionStartHeight returns the block height at which the session containing
// queryHeight starts, given the passed shared on-chain parameters.
// Returns 0 if the block height is not a consensus produced block.
// Example: If NumBlocksPerSession == 4, sessions start at blocks 1, 5, 9, etc.
func GetSessionStartHeight(sharedParams *sharedtypes.Params, queryHeight int64) int64 {
	if queryHeight <= 0 {
		return 0
	}

	numBlocksPerSession := int64(sharedParams.GetNumBlocksPerSession())

	// TODO_BLOCKER(#543): If the num_blocks_per_session param has ever been changed,
	// this function may cause unexpected behavior.
	return queryHeight - ((queryHeight - 1) % numBlocksPerSession)
}

// GetSessionEndHeight returns the block height at which the session containing
// queryHeight ends, given the passed shared on-chain parameters.
// Returns 0 if the block height is not a consensus produced block.
// Example: If NumBlocksPerSession == 4, sessions end at blocks 4, 8, 11, etc.
func GetSessionEndHeight(sharedParams *sharedtypes.Params, queryHeight int64) int64 {
	if queryHeight <= 0 {
		return 0
	}

	numBlocksPerSession := int64(sharedParams.GetNumBlocksPerSession())

	return GetSessionStartHeight(sharedParams, queryHeight) + numBlocksPerSession - 1
}

// GetSessionNumber returns the session number of the session containing queryHeight,
// given the passed on-chain shared parameters.
// shared on-chain parameters.
// Returns session number 0 if the block height is not a consensus produced block.
// Returns session number 1 for block 1 to block NumBlocksPerSession - 1 (inclusive).
// i.e. If NubBlocksPerSession == 4, session == 1 for [1, 4], session == 2 for [5, 8], etc.
func GetSessionNumber(sharedParams *sharedtypes.Params, queryHeight int64) int64 {
	if queryHeight <= 0 {
		return 0
	}

	numBlocksPerSession := int64(sharedParams.GetNumBlocksPerSession())
	// TODO_BLOCKER(#543): If the num_blocks_per_session param has ever been
	// changed, this function may cause unexpected behavior.
	return ((queryHeight - 1) / numBlocksPerSession) + 1
}

// GetSessionGracePeriodEndHeight returns the block height at which the grace period
// for the session ending with sessionEndHeight elapses.
func GetSessionGracePeriodEndHeight(sessionEndHeight int64) int64 {
	return sessionEndHeight + SessionGracePeriodBlocks
}

// IsGracePeriodElapsed returns true if the grace period for the session ending with
// sessionEndHeight has elapsed, given currentHeight.
func IsGracePeriodElapsed(sessionEndHeight, currentHeight int64) bool {
	return currentHeight > GetSessionGracePeriodEndHeight(sessionEndHeight)
}

// GetClaimWindowOpenHeight returns the block height at which the claim window of
// the session that includes queryHeight opens, given the passed sharedParams.
func GetClaimWindowOpenHeight(sharedParams *sharedtypes.Params, queryHeight int64) int64 {
	sessionEndHeight := GetSessionEndHeight(sharedParams, queryHeight)

	// An additional block is added to permit to relays arriving at the last block
	// of the session to be included in the claim before the smt is closed.
	sessionGracePeriodEndHeight := GetSessionGracePeriodEndHeight(sessionEndHeight)
	claimWindowOpenOffsetBlocks := int64(sharedParams.GetClaimWindowOpenOffsetBlocks())
	return claimWindowOpenOffsetBlocks + sessionGracePeriodEndHeight + 1
}
