package protocol

import "math/bits"

// TODO_BLOCKER: Revisit this part of the algorithm after initial TestNet Launch.
// TODO_TEST: Add extensive tests for the core relay mining business logic.

// MustCountDifficultyBits returns the number of leading zero bits in the given
// byte slice. It panics if an error is encountered.
func MustCountDifficultyBits(bz []byte) int {
	diff, err := CountDifficultyBits(bz)
	if err != nil {
		panic(err)
	}

	return diff
}

// CountDifficultyBits returns the number of leading zero bits in the given byte
// slice. It returns an error if the byte slice is all zero bits.
// TODO_IMPROVE(@h5law): Remove the forloop logic and replace with a simplified
// single method that accounts for the fact that paths are **always** 32 bytes.
//
//	if (lenBz) != 32 {
//	    return 0, ErrDifficulty.Wrapf("invalid byte length; got %d, want 32")
//	}
//	return bits.LeadingZeros64(binary.LittleEndian.Uint64(bz)), nil
func CountDifficultyBits(bz []byte) (int, error) {
	bzLen := len(bz)

	var zeroBits int
	for byteIdx, byteValue := range bz {
		if byteValue != 0 {
			zeroBits = bits.LeadingZeros8(byteValue)
			if zeroBits == 8 {
				// we already checked that byteValue != 0.
				return 0, ErrDifficulty.Wrap("impossible code path")
			}

			// We have byteIdx bytes that are all 0s and one byte that has
			// zeroBits number of leading 0 bits.
			return (byteIdx)*8 + zeroBits, nil
		}
	}

	return 0, ErrDifficulty.Wrapf("difficulty matches bytes length: %d; bytes (hex): % x", bzLen, bz)
}
