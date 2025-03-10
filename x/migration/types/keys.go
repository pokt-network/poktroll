package types

const (
	// ModuleName defines the module name
	ModuleName = "migration"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_migration"
)

var (
	ParamsKey = []byte("p_migration")
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}
