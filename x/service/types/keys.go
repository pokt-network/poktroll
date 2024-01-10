package types

import "bytes"

const (
	// ModuleName defines the module name
	ModuleName = "service"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_service"
)

// KeyDelimiter is the delimiter for composite keys.
var KeyDelimiter = []byte("/")

// KeyPrefix returns the given prefix as a byte slice for use with the KVStore.
func KeyPrefix(p string) []byte {
	return []byte(p)
}

// KeyComposite combines the given keys into a single key for use with KVStore.
func KeyComposite(keys ...[]byte) []byte {
	return bytes.Join(keys, KeyDelimiter)
}
