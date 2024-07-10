package rand

import (
	"bytes"
	"encoding/binary"
	"math/rand"

	"github.com/cometbft/cometbft/crypto"
)

// SeededInt63 generates a deterministic non-negative int64 by seeding a random
// source with the hash of seedParts.
//
// TODO_MAINNET: To support other language implementations of the protocol, the
// pseudo-random number generator used here should be language-agnostic (i.e. not
// golang specific).
func SeededInt63(seedParts ...[]byte) int64 {
	seedHashInputBz := bytes.Join(append([][]byte{}, seedParts...), nil)
	seedHash := crypto.Sha256(seedHashInputBz)
	seed, _ := binary.Varint(seedHash)

	return rand.NewSource(seed).Int63()
}
