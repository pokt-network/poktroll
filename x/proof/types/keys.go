package types

import (
	"bytes"
	"encoding/binary"
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
