package types

import "bytes"

const (
	// ModuleName defines the module name
	ModuleName = "application"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_application"
)

// KeyDelimiter is the delimiter for composite keys.
var KeyDelimiter = []byte("/")

// KeyPrefix returns the given prefix as a byte slice for use with the KVStore.
func KeyPrefix(prefix string) []byte {
	return []byte(prefix)
}

// KeyComposite combines the given keys into a single key for use with KVStore.
func KeyComposite(keys ...[]byte) []byte {
	return bytes.Join(keys, KeyDelimiter)
}
