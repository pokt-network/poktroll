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

func GetPathForProof(blockHash []byte, sessionId string) []byte {
	hasher := newHasher()
	_, err := hasher.Write(append(blockHash, []byte(sessionId)...))
	if err != nil {
		panic(err)
	}

	return hasher.Sum(nil)
}
