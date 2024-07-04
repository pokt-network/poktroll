package protocol

import (
	"encoding/binary"
	"math/bits"
)

// CountHashDifficultyBits returns the number of leading zero bits in the given byte
// slice. It returns an error if the byte slice is all zero bits.
func CountHashDifficultyBits(bz [32]byte) int {
	// Using BigEndian for  consistent bit/byte ordering such leading zeros
	// accumulate across adjacent bytes.
	// E.g.: 0x00, 0x255, 0x00, 0x00 has 8 leading zero bits. If LittleEndian
	// were applied instead, it would have 16 leading zeros because it would
	// look like 0x00, 0x00, 0x255, 0x00.
	return bits.LeadingZeros64(binary.BigEndian.Uint64(bz[:]))
}
