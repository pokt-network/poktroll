package types

const (
	// ModuleName defines the module name
	ModuleName = "supplier"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_supplier"
)

var (
	ParamsKey = []byte("p_supplier")
)

func KeyPrefix(p string) []byte { return []byte(p) }
