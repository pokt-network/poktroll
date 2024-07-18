package types

import (
	"encoding/binary"

	"github.com/pokt-network/poktroll/proto/types/tokenomics"
)

var _ binary.ByteOrder

const (
	// RelayMiningDifficultyKeyPrefix is the prefix to retrieve all RelayMiningDifficulty
	RelayMiningDifficultyKeyPrefix = "RelayMiningDifficulty/value/"
)

// RelayMiningDifficultyKey returns the store key to retrieve a RelayMiningDifficulty from the index fields
func RelayMiningDifficultyKey(
	serviceId string,
) []byte {
	return tokenomics.RelayMiningDifficultyKey(serviceId)
}
