package types

import "encoding/binary"

const (
	// ModuleName defines the module name
	ModuleName = "session"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_session"

	// TransientStoreKey is the transient store key used by test factories.
	// The app's key is provided by depinject and named "transient:session".
	// Transient stores are block-scoped: part of the multistore branch txs
	// execute against, wiped on Commit. Backs the hydrated-session memo
	// (see keeper/session_memo.go).
	TransientStoreKey = "transient_session"
)

var (
	ParamsKey = []byte("p_session")

	// ParamsHistoryKeyPrefix is the prefix for storing historical session params.
	// Key format: ParamsHistoryKeyPrefix | BigEndian(effectiveHeight)
	// This enables efficient range queries to find params effective at a given height.
	ParamsHistoryKeyPrefix = []byte("session_params_history/")

	// SessionMemoKeyPrefix is the transient-store prefix for hydrated sessions
	// memoized within a block. See SessionMemoKey for the key format.
	SessionMemoKeyPrefix = []byte("session_memo/")
)

// SessionMemoKey returns the transient-store key under which the session of
// (appAddr, serviceId) requested at blockHeight is memoized within a block:
// appAddr | "/" | serviceId | "/" | BigEndian(blockHeight).
// Neither component may contain "/" (bech32 alphabet, service ID charset), and
// the height is a fixed-width suffix, so keys are unambiguous.
func SessionMemoKey(appAddr, serviceId string, blockHeight int64) []byte {
	key := []byte(appAddr + "/" + serviceId + "/")
	return binary.BigEndian.AppendUint64(key, uint64(blockHeight))
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
