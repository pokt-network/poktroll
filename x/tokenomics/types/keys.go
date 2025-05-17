package types

import "encoding/binary"

const (
	// ModuleName defines the module name
	ModuleName = "tokenomics"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_tokenomics"

	// ParamsUpdateKeyPrefix defines the prefix for params updates.
	// This is used to store params updates at a specific height.
	ParamsUpdateKeyPrefix = "tokenomics_params_update/effective_height/"
)

var ParamsKey = []byte("p_tokenomics")

func KeyPrefix(p string) []byte { return []byte(p) }

// IntKey converts an integer value to a byte slice for use in store keys
// Appends a '/' separator to the end of the key for consistent prefix scanning
func IntKey(intIndex int64) []byte {
	var key []byte

	heightBz := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBz, uint64(intIndex))
	key = append(key, heightBz...)
	key = append(key, []byte("/")...)

	return key
}
