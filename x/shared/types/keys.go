package types

import "encoding/binary"

const (
	// ModuleName defines the module name
	ModuleName = "shared"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_shared"
)

var (
	ParamsKey = []byte("p_shared")
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}

// IntKey converts an integer value to a byte slice for use in store keys.
// Appends a '/' separator to the end of the key for consistent prefix scanning
func IntKey(intIndex int64) []byte {
	var key []byte

	intIndexBz := make([]byte, 8)
	binary.BigEndian.PutUint64(intIndexBz, uint64(intIndex))
	key = append(key, intIndexBz...)
	key = append(key, []byte("/")...)

	return key
}

// StringKey converts a string value to a byte slice for use in store keys
// Appends a '/' separator to the end of the key for consistent prefix scanning
func StringKey(strIndex string) []byte {
	var key []byte

	strIndexBz := []byte(strIndex)
	key = append(key, strIndexBz...)
	key = append(key, []byte("/")...)

	return key
}
