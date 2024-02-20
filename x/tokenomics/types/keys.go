package types

const (
	// ModuleName defines the module name
	ModuleName = "tokenomics"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_tokenomics"
)

var (
	ParamsKey = []byte("p_tokenomics")
)

func KeyPrefix(p string) []byte { return []byte(p) }
