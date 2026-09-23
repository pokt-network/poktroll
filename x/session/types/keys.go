package types

import "encoding/binary"

const (
	// ModuleName defines the module name
	ModuleName = "session"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_session"

	// TransientStoreKey is the name of the session module's transient store key.
	// It MUST match the name depinject gives it ("transient:<module>", see
	// runtime.ProvideTransientStoreKey) so test factories and the app mount the
	// same store. Transient stores are block-scoped: part of the multistore
	// branch txs execute against, wiped on Commit. Backs the session supplier
	// memo (see keeper/session_memo.go).
	TransientStoreKey = "transient:" + ModuleName
)

var (
	ParamsKey = []byte("p_session")

	// ParamsHistoryKeyPrefix is the prefix for storing historical session params.
	// Key format: ParamsHistoryKeyPrefix | BigEndian(effectiveHeight)
	// This enables efficient range queries to find params effective at a given height.
	ParamsHistoryKeyPrefix = []byte("session_params_history/")

	// SessionMemoKeyPrefix is the transient-store prefix for session supplier
	// selections memoized within a block. See SessionMemoKey for the key format.
	SessionMemoKeyPrefix = []byte("session_memo/")
)

// SessionMemoKey returns the transient-store key under which a session's
// supplier selection is memoized within a block. It holds every input of the
// selection, so a key can only match a selection computed from the same inputs:
// BigEndian(currentHeight) | BigEndian(queryHeight) | BigEndian(numSuppliersPerSession) | sessionIDBz.
// currentHeight limits a hit to the block that wrote it, even when a context
// moves to a new height without a Commit. sessionIDBz (block hash, service ID,
// app address, session start height) seeds the supplier weights. The fixed-width
// fields come first, so the variable-length tail keeps keys unambiguous.
func SessionMemoKey(currentHeight, queryHeight int64, numSuppliersPerSession uint64, sessionIDBz []byte) []byte {
	key := make([]byte, 0, 24+len(sessionIDBz))
	key = binary.BigEndian.AppendUint64(key, uint64(currentHeight))
	key = binary.BigEndian.AppendUint64(key, uint64(queryHeight))
	key = binary.BigEndian.AppendUint64(key, numSuppliersPerSession)
	return append(key, sessionIDBz...)
}

func KeyPrefix(p string) []byte { return []byte(p) }

// ParamsHistoryKey returns the store key for session params at a given effective height.
// Uses big-endian encoding to ensure lexicographic ordering matches numeric ordering.
func ParamsHistoryKey(effectiveHeight int64) []byte {
	heightBytes := make([]byte, 8)
	// Use big-endian so keys are ordered by height when iterating
	binary.BigEndian.PutUint64(heightBytes, uint64(effectiveHeight))
	return append(ParamsHistoryKeyPrefix, heightBytes...)
}
