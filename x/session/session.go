package session

import (
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
)

func GetSessionGracePeriodBlockCount(numBlocksPerSession uint64) uint64 {
	return sessionkeeper.SessionGracePeriod * numBlocksPerSession
}

func IsWithinGracePeriod(numBlocksPerSession uint64, sessionEndHeight, queryHeight int64) bool {
	sessionGracePeriodBlockCount := GetSessionGracePeriodBlockCount(numBlocksPerSession)
	sessionGracePeriodEndHeight := sessionEndHeight + int64(sessionGracePeriodBlockCount)
	return queryHeight <= sessionGracePeriodEndHeight
}

func IsPastGracePeriod(numBlocksPerSession uint64, sessionEndHeight, queryHeight int64) bool {
	sessionGracePeriodBlockCount := GetSessionGracePeriodBlockCount(numBlocksPerSession)
	sessionGracePeriodEndHeight := sessionEndHeight + int64(sessionGracePeriodBlockCount)
	return queryHeight > sessionGracePeriodEndHeight
}
