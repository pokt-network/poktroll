package rand

import (
	"bytes"
	"encoding/binary"
	"math/rand"

	"github.com/cometbft/cometbft/crypto"
)

// SeededFloat32 generates a deterministic float32 between 0 and 1 given a seed.
//
// TODO_MAINNET: To support other language implementations of the protocol, the
// pseudo-random number generator used here should be language-agnostic (i.e. not
// golang specific).
func SeededFloat32(seedParts ...[]byte) (float32, error) {
	seedHashInputBz := bytes.Join(append([][]byte{}, seedParts...), nil)
	seedHash := crypto.Sha256(seedHashInputBz)
	seed, _ := binary.Varint(seedHash)

	// Construct a pseudo-random number generator with the seed.
	pseudoRand := rand.New(rand.NewSource(seed))

	// Generate a random uint32.
	randUint32 := pseudoRand.Uint32()

	// Clamp the random float32 between [0,1]. This is achieved by dividing the random uint32
	// by the most significant digit of a float32, which is 2^32, guaranteeing an output between
	// 0 and 1, inclusive.
	oneMostSignificantDigitFloat32 := float32(1 << 32)
	randClampedFloat32 := float32(randUint32) / oneMostSignificantDigitFloat32

	return randClampedFloat32, nil
}
