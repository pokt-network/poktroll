package protocol

import (
	"encoding/binary"
	"math/bits"
)

// CountHashDifficultyBits returns the number of leading zero bits in the given byte slice.
// TODO_MAINNET: Consider generalizing difficulty to a target hash. See:
// - https://bitcoin.stackexchange.com/questions/107976/bitcoin-difficulty-why-leading-0s
// - https://bitcoin.stackexchange.com/questions/121920/is-it-always-possible-to-find-a-number-whose-hash-starts-with-a-certain-number-o
// - https://github.com/pokt-network/poktroll/pull/656/files#r1666712528
func CountHashDifficultyBits(bz [32]byte) int {
	// Using BigEndian for contiguous bit/byte ordering such leading zeros
	// accumulate across adjacent bytes.
	// E.g.: []byte{0, 0b00111111, 0x00, 0x00} has 10 leading zero bits. If
	// LittleEndian were applied instead, it would have 18 leading zeros because it would
	// look like []byte{0, 0, 0b00111111, 0}.
	return bits.LeadingZeros64(binary.BigEndian.Uint64(bz[:]))
}
