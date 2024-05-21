package session

import (
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
)

// GetSessionGracePeriodBlockCount returns the number of blocks in the grace period
// given some numBlocksPerSession.
func GetSessionGracePeriodBlockCount(numBlocksPerSession uint64) uint64 {
	return sessionkeeper.SessionGracePeriod * numBlocksPerSession
}

// IsWithinGracePeriod returns true if the grace period for the session ending with
// sessionEndHeight has not yet elapsed, given currentHeight.
func IsWithinGracePeriod(numBlocksPerSession uint64, sessionEndHeight, currentHeight int64) bool {
	sessionGracePeriodBlockCount := GetSessionGracePeriodBlockCount(numBlocksPerSession)
	sessionGracePeriodEndHeight := sessionEndHeight + int64(sessionGracePeriodBlockCount)
	return currentHeight <= sessionGracePeriodEndHeight
}

// IsPastGracePeriod returns true if the grace period for the session ending with
// sessionEndHeight has elapsed, given currentHeight.
func IsPastGracePeriod(numBlocksPerSession uint64, sessionEndHeight, currentHeight int64) bool {
	sessionGracePeriodBlockCount := GetSessionGracePeriodBlockCount(numBlocksPerSession)
	sessionGracePeriodEndHeight := sessionEndHeight + int64(sessionGracePeriodBlockCount)
	return currentHeight > sessionGracePeriodEndHeight
}
