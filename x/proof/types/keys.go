package types

const (
	// ModuleName defines the module name
	ModuleName = "proof"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_proof"
)

var (
	ParamsKey = []byte("p_proof")
)

func KeyPrefix(p string) []byte { return []byte(p) }
