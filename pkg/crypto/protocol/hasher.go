package protocol

import "crypto/sha256"

const (
	TrieHasherSize = sha256.Size
	TrieRootSize   = TrieHasherSize + trieRootMetadataSize
	// TODO_CONSIDERATION: Export this from the SMT package.
	trieRootMetadataSize = 16
)

var NewTrieHasher = sha256.New
