package protocol

import (
	"crypto/sha256"

	"github.com/pokt-network/smt"
)

// newHasher is the hash function used by the SMT specification.
var newHasher = sha256.New

// GetPathForProof computes the path to be used for proof validation by hashing
// the block hash and session id.
func GetPathForProof(blockHash []byte, sessionId string) []byte {
	hasher := newHasher()
	if _, err := hasher.Write(append(blockHash, []byte(sessionId)...)); err != nil {
		panic(err)
	}

	return hasher.Sum(nil)
}

// NewSMTSpec returns the SMT specification used for the proof verification.
// It uses a new hasher at every call to avoid concurrency issues that could be
// caused by a shared hasher.
func NewSMTSpec() *smt.TrieSpec {
	trieSpec := smt.NewTrieSpec(
		newHasher(), true,
		smt.WithValueHasher(nil),
	)

	return &trieSpec
}
