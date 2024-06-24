package types

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
