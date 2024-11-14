package rand

import (
	"bytes"
	"encoding/binary"
	"math/rand"

	"github.com/cometbft/cometbft/crypto"
)

// SeededFloat64 generates a deterministic float64 between 0 and 1 given a seed.
//
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
