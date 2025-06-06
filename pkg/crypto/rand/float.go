package rand

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"math/rand"

	"github.com/cometbft/cometbft/crypto"
)

// SeededFloat64 generates a deterministic float64 between 0 and 1 given a seed.

// TODO_MAINNET: To support other language implementations of the protocol, the
// pseudo-random number generator used here should be language-agnostic (i.e. not
// golang specific).
func SeededFloat64(seedParts ...[]byte) float64 {
	seedHashInputBz := bytes.Join(append([][]byte{}, seedParts...), nil)
	seedHash := crypto.Sha256(seedHashInputBz)
	seed, _ := binary.Varint(seedHash)

	// Construct a pseudo-random number generator with the seed.
	pseudoRand := rand.New(rand.NewSource(seed))

	return pseudoRand.Float64()
}

func DeterministicFloat64(seedParts ...[]byte) float64 {
	// Join seed parts
	seedBytes := bytes.Join(seedParts, nil)
	// Hash to fixed 32-byte output
	hash := sha256.Sum256(seedBytes)
	// Use first 8 bytes as uint64
	num := binary.BigEndian.Uint64(hash[0:8])
	// Normalize to [0,1)
	return float64(num) / float64(^uint64(0))
}
