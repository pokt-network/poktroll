package session

import (
	"github.com/pokt-network/poktroll/x/session/keeper"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var sharedParamsHistory = sharedtypes.InitialParamsHistory(sharedtypes.DefaultParams())

// GetSessionIdWithDefaultParams returns the string and bytes representation of the
// sessionId for the session containing blockHeight, given the default shared onchain
// parameters, application public key, service ID, and block hash.
func GetSessionIdWithDefaultParams(
	appPubKey,
	serviceId string,
	blockHashBz []byte,
	blockHeight int64,
) (sessionId string, sessionIdBz []byte) {
	return keeper.GetSessionId(sharedParamsHistory, appPubKey, serviceId, blockHashBz, blockHeight)
}

// GetSessionStartHeightWithDefaultParams returns the block height at which the
// session containing queryHeight starts, given the default shared onchain
// parameters.
// See shared.GetSessionStartHeight for more details.
func GetSessionStartHeightWithDefaultParams(queryHeight int64) int64 {
	return sharedParamsHistory.GetSessionStartHeight(queryHeight)
}

// GetSessionEndHeightWithDefaultParams returns the block height at which the session
// containing queryHeight ends, given the default shared onchain parameters.
// See shared.GetSessionEndHeight for more details.
func GetSessionEndHeightWithDefaultParams(queryHeight int64) int64 {
	return sharedParamsHistory.GetSessionEndHeight(queryHeight)
}

// GetSessionNumberWithDefaultParams returns the session number of the session
// containing queryHeight, given the default onchain shared parameters.
// See shared.GetSessionNumber for more details.
func GetSessionNumberWithDefaultParams(queryHeight int64) int64 {
	return sharedParamsHistory.GetSessionNumber(queryHeight)
}
