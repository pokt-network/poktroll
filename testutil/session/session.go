package session

import (
	"github.com/pokt-network/poktroll/x/session/keeper"
	"github.com/pokt-network/poktroll/x/shared"
	"github.com/pokt-network/poktroll/x/shared/types"
)

// GetDefaultSessionId returns the string and bytes representation of the sessionId
// given the application public key, service ID, block hash, and block height
// that is used to get the session start block height.
func GetDefaultSessionId(
	appPubKey,
	serviceId string,
	blockHashBz []byte,
	blockHeight int64,
) (sessionId string, sessionIdBz []byte) {
	sharedParams := types.DefaultParams()
	return keeper.GetSessionId(&sharedParams, appPubKey, serviceId, blockHashBz, blockHeight)
}

// GetDefaultSessionStartHeight returns the block height at which the session starts
// given the default shared on-chain parameters.
// Returns 0 if the block height is not a consensus produced block.
// Example: If NumBlocksPerSession == 4, sessions start at blocks 1, 5, 9, etc.
func GetDefaultSessionStartHeight(queryHeight int64) int64 {
	sharedParams := types.DefaultParams()
	return shared.GetSessionStartHeight(&sharedParams, queryHeight)
}

// GetDefaultSessionEndHeight returns the block height at which the session ends,
// given the default shared on-chain parameters.
// Returns 0 if the block height is not a consensus produced block.
// Example: If NumBlocksPerSession == 4, sessions end at blocks 4, 8, 11, etc.
func GetDefaultSessionEndHeight(queryHeight int64) int64 {
	sharedParams := types.DefaultParams()
	return shared.GetSessionEndHeight(&sharedParams, queryHeight)
}

// GetDefaultSessionNumber returns the session number given the block height given the
// default on-chain shared parameters.
// Returns session number 0 if the block height is not a consensus produced block.
// Returns session number 1 for block 1 to block NumBlocksPerSession - 1 (inclusive).
// i.e. If NubBlocksPerSession == 4, session == 1 for [1, 4], session == 2 for [5, 8], etc.
func GetDefaultSessionNumber(queryHeight int64) int64 {
	sharedParams := types.DefaultParams()
	return shared.GetSessionNumber(&sharedParams, queryHeight)
}
