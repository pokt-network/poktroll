package session

import (
	"github.com/pokt-network/poktroll/x/session/keeper"
	"github.com/pokt-network/poktroll/x/shared"
	"github.com/pokt-network/poktroll/x/shared/types"
)

// GetSessionIdWithDefaultParams returns the string and bytes representation of the
// sessionId for the session containing blockHeight, given the default shared on-chain
// parameters, application public key, service ID, and block hash.
func GetSessionIdWithDefaultParams(
	appPubKey,
	serviceId string,
	blockHashBz []byte,
	blockHeight int64,
) (sessionId string, sessionIdBz []byte) {
	sharedParams := types.DefaultParams()
	return keeper.GetSessionId(&sharedParams, appPubKey, serviceId, blockHashBz, blockHeight)
}

// GetSessionStartHeightWithDefaultParams returns the block height at which the
// session containing queryHeight starts, given the default shared on-chain
// parameters.
// Returns 0 if the block height is not a consensus produced block.
// Example: If NumBlocksPerSession == 4, sessions start at blocks 1, 5, 9, etc.
func GetSessionStartHeightWithDefaultParams(queryHeight int64) int64 {
	sharedParams := types.DefaultParams()
	return shared.GetSessionStartHeight(&sharedParams, queryHeight)
}

// GetSessionEndHeightWithDefaultParams returns the block height at which the session
// containing queryHeight ends, given the default shared on-chain parameters.
// Returns 0 if the block height is not a consensus produced block.
// Example: If NumBlocksPerSession == 4, sessions end at blocks 4, 8, 11, etc.
func GetSessionEndHeightWithDefaultParams(queryHeight int64) int64 {
	sharedParams := types.DefaultParams()
	return shared.GetSessionEndHeight(&sharedParams, queryHeight)
}

// GetSessionNumberWithDefaultParams returns the session number of the session
// containing queryHeight, given the default on-chain shared parameters.
// Returns session number 0 if the block height is not a consensus produced block.
// Returns session number 1 for block 1 to block NumBlocksPerSession - 1 (inclusive).
// i.e. If NubBlocksPerSession == 4, session == 1 for [1, 4], session == 2 for [5, 8], etc.
func GetSessionNumberWithDefaultParams(queryHeight int64) int64 {
	sharedParams := types.DefaultParams()
	return shared.GetSessionNumber(&sharedParams, queryHeight)
}
