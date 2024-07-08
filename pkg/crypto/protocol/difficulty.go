package protocol

import (
	"crypto/sha256"
	"encoding/hex"
	"math/big"
)

// Difficulty1Hash represents the "easiest" difficulty.
var (
	Difficulty1HashHex   = "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	Difficulty1HashBz, _ = hex.DecodeString(Difficulty1HashHex)
	Difficulty1HashInt   = new(big.Int).SetBytes(Difficulty1HashBz)
)

// GetDifficultyFromHash returns the "difficulty" of the given hash, with respect
// to the "highest" target hash, Difficulty1Hash.
// - https://bitcoin.stackexchange.com/questions/107976/bitcoin-difficulty-why-leading-0s
// - https://bitcoin.stackexchange.com/questions/121920/is-it-always-possible-to-find-a-number-whose-hash-starts-with-a-certain-number-o
func GetDifficultyFromHash(hashBz [sha256.Size]byte) int64 {
	hashInt := new(big.Int).SetBytes(hashBz[:])

	// difficulty is the ratio of the highest target hash to the given hash.
	return new(big.Int).Div(Difficulty1HashInt, hashInt).Int64()
}
