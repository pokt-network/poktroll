package protocol

import (
	"crypto/sha256"

	"github.com/pokt-network/smt"
)

// SMT specification used for the proof verification.
var (
	newHasher = sha256.New
	SmtSpec   smt.TrieSpec
)

func init() {
	// Use a spec that does not prehash values in the smst. This returns a nil value
	// hasher for the proof verification in order to avoid hashing the value twice.
	SmtSpec = smt.NewTrieSpec(
		newHasher(), true,
		smt.WithValueHasher(nil),
	)
}

// GetPathForProof computes the path to be used for proof validation by hashing
// the block hash and session id.
func GetPathForProof(blockHash []byte, sessionId string) []byte {
	hasher := newHasher()
	if _, err := hasher.Write(append(blockHash, []byte(sessionId)...)); err != nil {
		panic(err)
	}

	return hasher.Sum(nil)
}
