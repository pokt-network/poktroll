package rand

import (
	"bytes"
	"encoding/binary"

	"github.com/cometbft/cometbft/crypto"
)

// SeededFloat64 generates a deterministic float64 in [0, 1) given a seed.
//
// The value is the top 53 bits of the first 8 bytes of SHA-256(seed), scaled to
// [0, 1). It is uniform and reproducible in any language: no Go-specific PRNG.
func SeededFloat64(seedParts ...[]byte) float64 {
	seedHash := crypto.Sha256(bytes.Join(seedParts, nil))
	return float64(binary.BigEndian.Uint64(seedHash[:8])>>11) / (1 << 53)
}
