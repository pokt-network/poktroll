package protocol

import (
	"crypto/sha256"

	"github.com/pokt-network/smt"
)

const (
	RelayHasherSize      = sha256.Size
	TrieHasherSize       = sha256.Size
	TrieRootSize         = TrieHasherSize + trieRootMetadataSize
	TrieRootSumSize      = 8  // TODO_CONSIDERATION: Export this from the SMT package.
	trieRootMetadataSize = 16 // TODO_CONSIDERATION: Export this from the SMT package.

)

var (
	NewRelayHasher = sha256.New
	NewTrieHasher  = sha256.New
)

func SMTValueHasher() smt.TrieSpecOption {
	return smt.WithValueHasher(nil)
}
