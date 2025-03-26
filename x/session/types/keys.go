package types

import "strconv"

const (
	// ModuleName defines the module name
	ModuleName = "session"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_session"

	// ParamsUpdateKeyPrefix defines the prefix for params updates.
	// This is used to store params updates at a specific height.
	ParamsUpdateKeyPrefix = "session_params_update/effective_height/"
)

var ParamsKey = []byte("p_session")

func KeyPrefix(p string) []byte { return []byte(p) }

// ParamsUpdateKey returns the key for the params update at the given height.
func ParamsUpdateKey(height uint64) []byte {
	heightStr := strconv.FormatUint(height, 10)
	return []byte(ParamsUpdateKeyPrefix + heightStr)
}
