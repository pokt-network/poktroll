package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// RelayMiningDifficultyKeyPrefix is the prefix to retrieve all RelayMiningDifficulty
	RelayMiningDifficultyKeyPrefix = "RelayMiningDifficulty/value/"

	// RelayMiningDifficultyHistoryKeyPrefix is the prefix for storing historical difficulty.
	// Key format: RelayMiningDifficultyHistoryKeyPrefix | serviceId | "/" | BigEndian(effectiveHeight)
	// This enables efficient range queries to find difficulty effective at a given height.
	RelayMiningDifficultyHistoryKeyPrefix = "RelayMiningDifficulty/history/"
)

// RelayMiningDifficultyKey returns the store key to retrieve a RelayMiningDifficulty from the index fields
func RelayMiningDifficultyKey(
	serviceId string,
) []byte {
	var key []byte

	serviceIdBz := []byte(serviceId)
	key = append(key, serviceIdBz...)
	key = append(key, []byte("/")...)

	return key
}

// RelayMiningDifficultyHistoryKey returns the store key for difficulty at a given service and height.
// Uses big-endian encoding to ensure lexicographic ordering matches numeric ordering.
func RelayMiningDifficultyHistoryKey(serviceId string, effectiveHeight int64) []byte {
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, uint64(effectiveHeight))

	key := append([]byte(serviceId), []byte("/")...)
	key = append(key, heightBytes...)
	return append([]byte(RelayMiningDifficultyHistoryKeyPrefix), key...)
}

// RelayMiningDifficultyHistoryKeyPrefixForService returns the prefix for all history entries of a service.
func RelayMiningDifficultyHistoryKeyPrefixForService(serviceId string) []byte {
	key := append([]byte(serviceId), []byte("/")...)
	return append([]byte(RelayMiningDifficultyHistoryKeyPrefix), key...)
}
