package types

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
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
