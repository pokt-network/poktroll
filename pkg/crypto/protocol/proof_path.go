package protocol

import (
	"github.com/pokt-network/smt"
)

// GetPathForProof computes the path to be used for proof validation by hashing
// the block hash and session id.
func GetPathForProof(blockHash []byte, sessionId string) []byte {
	hasher := NewTrieHasher()
	if _, err := hasher.Write(append(blockHash, []byte(sessionId)...)); err != nil {
		panic(err)
	}

	return hasher.Sum(nil)
}

// NewSMTSpec returns the SMT specification used for proof verification.
// A new hasher is created for each call to prevent concurrency issues
// from shared state.
func NewSMTSpec() *smt.TrieSpec {
	trieSpec := smt.NewTrieSpec(
		NewTrieHasher(), true,
		SMTValueHasher(),
	)

	return &trieSpec
}
