package protocol

import (
	"encoding/hex"
	"math/big"
)

var (
	// Difficulty1HashBz is the chosen "highest" (easiest) target hash, which
	// corresponds to the lowest possible difficulty. It effectively calibrates
	// the difficulty number (which is returned by GetDifficultyFromHash) by defining
	// the hash which corresponds to difficulty 1.
	// - https://bitcoin.stackexchange.com/questions/107976/bitcoin-difficulty-why-leading-0s
	// - https://bitcoin.stackexchange.com/questions/121920/is-it-always-possible-to-find-a-number-whose-hash-starts-with-a-certain-number-o
	Difficulty1HashBz, _ = hex.DecodeString("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
)

// GetDifficultyFromHash returns the "difficulty" of the given hash, with respect
// to the "highest" (easiest) target hash, Difficulty1Hash.
func GetDifficultyFromHash(hashBz [RelayHasherSize]byte) int64 {
	difficulty1HashInt := new(big.Int).SetBytes(Difficulty1HashBz)
	hashInt := new(big.Int).SetBytes(hashBz[:])

	// difficulty is the ratio of the highest target hash to the given hash.
	return new(big.Int).Div(difficulty1HashInt, hashInt).Int64()
}
