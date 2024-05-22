package session

import (
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
)

// GetSessionGracePeriodBlockCount returns the number of blocks in the grace period
// given some numBlocksPerSession.
func GetSessionGracePeriodBlockCount(numBlocksPerSession uint64) uint64 {
	return sessionkeeper.SessionGracePeriod * numBlocksPerSession
}

// GetSessionGracePeriodEndHeight returns the height at which the grace period for
// the session ending with sessionEndHeight elapses.
func GetSessionGracePeriodEndHeight(numBlocksPerSession uint64, sessionEndHeight int64) int64 {
	sessionGracePeriodBlockCount := GetSessionGracePeriodBlockCount(numBlocksPerSession)
	return sessionEndHeight + int64(sessionGracePeriodBlockCount)
}

// IsWithinGracePeriod returns true if the grace period for the session ending with
// sessionEndHeight has not yet elapsed, given currentHeight.
func IsWithinGracePeriod(numBlocksPerSession uint64, sessionEndHeight, currentHeight int64) bool {
	sessionGracePeriodEndHeight := GetSessionGracePeriodEndHeight(numBlocksPerSession, sessionEndHeight)
	return currentHeight <= sessionGracePeriodEndHeight
}

// IsPastGracePeriod returns true if the grace period for the session ending with
// sessionEndHeight has elapsed, given currentHeight.
func IsPastGracePeriod(numBlocksPerSession uint64, sessionEndHeight, currentHeight int64) bool {
	sessionGracePeriodEndHeight := GetSessionGracePeriodEndHeight(numBlocksPerSession, sessionEndHeight)
	return currentHeight > sessionGracePeriodEndHeight
}
