package session

import (
	"github.com/pokt-network/poktroll/x/session/keeper"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// GetSessionIdWithDefaultParams returns the string and bytes representation of the
// sessionId for the session containing blockHeight, given the default shared onchain
// parameters, application public key, service ID, and block hash.
func GetSessionIdWithDefaultParams(
	appPubKey,
	serviceId string,
	blockHashBz []byte,
	blockHeight int64,
) (sessionId string, sessionIdBz []byte) {
	sharedParams := sharedtypes.DefaultParams()
	return keeper.GetSessionId(&sharedParams, appPubKey, serviceId, blockHashBz, blockHeight)
}

// GetSessionStartHeightWithDefaultParams returns the block height at which the
// session containing queryHeight starts, given the default shared onchain
// parameters.
// See shared.GetSessionStartHeight for more details.
func GetSessionStartHeightWithDefaultParams(queryHeight int64) int64 {
	sharedParams := sharedtypes.DefaultParams()
	return sharedtypes.GetSessionStartHeight(&sharedParams, queryHeight)
}

// GetSessionEndHeightWithDefaultParams returns the block height at which the session
// containing queryHeight ends, given the default shared onchain parameters.
// See shared.GetSessionEndHeight for more details.
func GetSessionEndHeightWithDefaultParams(queryHeight int64) int64 {
	sharedParams := sharedtypes.DefaultParams()
	return sharedtypes.GetSessionEndHeight(&sharedParams, queryHeight)
}

// GetSessionNumberWithDefaultParams returns the session number of the session
// containing queryHeight, given the default onchain shared parameters.
// See shared.GetSessionNumber for more details.
func GetSessionNumberWithDefaultParams(queryHeight int64) int64 {
	sharedParams := sharedtypes.DefaultParams()
	return sharedtypes.GetSessionNumber(&sharedParams, queryHeight)
}
