package types

import "strconv"

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

// ParamsUpdateKey returns the key for the params update at the given height.
func ParamsUpdateKey(height uint64) []byte {
	heightStr := strconv.FormatUint(height, 10)
	return []byte(ParamsUpdateKeyPrefix + heightStr)
}
