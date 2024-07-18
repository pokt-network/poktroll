package tokenomics

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
