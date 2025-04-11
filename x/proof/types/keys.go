package types

import (
	"bytes"
	"strconv"
)

const (
	// ModuleName defines the module name
	ModuleName = "proof"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_proof"

	// ParamsUpdateKeyPrefix defines the prefix for params updates.
	// This is used to store params updates at a specific height.
	ParamsUpdateKeyPrefix = "proof_params_update/effective_height/"
)

var (
	ParamsKey = []byte("p_proof")
	// KeyDelimiter is the delimiter for composite keys.
	KeyDelimiter = []byte("/")
)

func KeyPrefix(p string) []byte { return []byte(p) }

// KeyComposite combines the given keys into a single key for use with KVStore.
func KeyComposite(keys ...[]byte) []byte {
	return bytes.Join(keys, KeyDelimiter)
}

// ParamsUpdateKey returns the key for the params update at the given height.
func ParamsUpdateKey(height uint64) []byte {
	heightStr := strconv.FormatUint(height, 10)
	return []byte(ParamsUpdateKeyPrefix + heightStr)
}
