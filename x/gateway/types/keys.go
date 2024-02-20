package types

const (
	// ModuleName defines the module name
	ModuleName = "gateway"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_gateway"
)

var ParamsKey = []byte("p_gateway")

func KeyPrefix(p string) []byte { return []byte(p) }
