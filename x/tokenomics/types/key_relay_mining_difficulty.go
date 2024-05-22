package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// ClaimPrimaryKeyPrefix is the prefix to retrieve the entire Claim object (the primary store)
	RelayMiningDifficultyKeyPrefix = "RelayMiningDifficulty/service_id/"
)

// ClaimPrimaryKey returns the primary store key used to retrieve a Claim by creating a composite key of the sessionId and supplierAddr.
func RelayMiningDifficultyKey(serviceId string) []byte {
	// We are guaranteed uniqueness of the primary key if it's a composite of the (sessionId, supplierAddr)
	// because every supplier can only have one claim per session.
	return KeyComposite([]byte(serviceId))
}
