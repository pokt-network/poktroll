package protocol

import "crypto/sha256"

const (
	RelayHasherSize = sha256.Size
	TrieHasherSize = sha256.Size
	TrieRootSize   = TrieHasherSize + trieRootMetadataSize
	// TODO_CONSIDERATION: Export this from the SMT package.
	trieRootMetadataSize = 16
)

var (
	NewRelayHasher = sha256.New
	NewTrieHasher = sha256.New
)
