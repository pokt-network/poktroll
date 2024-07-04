package protocol

import (
	"encoding/binary"
	"math/bits"
)

// CountHashDifficultyBits returns the number of leading zero bits in the given byte
// slice. It returns an error if the byte slice is all zero bits.
func CountHashDifficultyBits(bz [32]byte) int {
	// Using BigEndian for contiguous bit/byte ordering such leading zeros
	// accumulate across adjacent bytes.
	// E.g.: []byte{0, 0b00111111, 0x00, 0x00} has 10 leading zero bits. If
	// LittleEndian were applied instead, it would have 18 leading zeros because it would
	// look like []byte{0, 0, 0b00111111, 0}.
	return bits.LeadingZeros64(binary.BigEndian.Uint64(bz[:]))
}
