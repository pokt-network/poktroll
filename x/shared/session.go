package shared

const NumBlocksPerSession = 4

// SessionGracePeriodBlocks SHOULD be a multiple of
const SessionGracePeriodBlocks = 4

// GetSessionStartBlockHeight returns the block height at which the session starts
// Returns 0 if the block height is not a consensus produced block.
// Example: If NumBlocksPerSession == 4, sessions start at blocks 1, 5, 9, etc.
func GetSessionStartBlockHeight(blockHeight int64) int64 {
	if blockHeight <= 0 {
		return 0
	}

	// TODO_BLOCKER(#543): If the num_blocks_per_session param has ever been changed,
	// this function may cause unexpected behavior.
	return blockHeight - ((blockHeight - 1) % NumBlocksPerSession)
}

// GetSessionEndBlockHeight returns the block height at which the session ends
// Returns 0 if the block height is not a consensus produced block.
// Example: If NumBlocksPerSession == 4, sessions end at blocks 4, 8, 11, etc.
func GetSessionEndBlockHeight(blockHeight int64) int64 {
	if blockHeight <= 0 {
		return 0
	}

	return GetSessionStartBlockHeight(blockHeight) + NumBlocksPerSession - 1
}

// GetSessionNumber returns the session number given the block height.
// Returns session number 0 if the block height is not a consensus produced block.
// Returns session number 1 for block 1 to block NumBlocksPerSession - 1 (inclusive).
// i.e. If NubBlocksPerSession == 4, session == 1 for [1, 4], session == 2 for [5, 8], etc.
func GetSessionNumber(blockHeight int64) int64 {
	if blockHeight <= 0 {
		return 0
	}

	// TODO_BLOCKER(#543): If the num_blocks_per_session param has ever been changed,
	// this function may cause unexpected behavior.
	return ((blockHeight - 1) / NumBlocksPerSession) + 1
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
