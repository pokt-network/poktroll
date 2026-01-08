package types

import "encoding/binary"

const (
	// ModuleName defines the module name
	ModuleName = "session"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_session"
)

var (
	ParamsKey = []byte("p_session")

	// ParamsHistoryKeyPrefix is the prefix for storing historical session params.
	// Key format: ParamsHistoryKeyPrefix | BigEndian(effectiveHeight)
	// This enables efficient range queries to find params effective at a given height.
	ParamsHistoryKeyPrefix = []byte("params_history/")
)

func KeyPrefix(p string) []byte { return []byte(p) }

// ParamsHistoryKey returns the store key for session params at a given effective height.
// Uses big-endian encoding to ensure lexicographic ordering matches numeric ordering.
func ParamsHistoryKey(effectiveHeight int64) []byte {
	heightBytes := make([]byte, 8)
	// Use big-endian so keys are ordered by height when iterating
	binary.BigEndian.PutUint64(heightBytes, uint64(effectiveHeight))
	return append(ParamsHistoryKeyPrefix, heightBytes...)
}
