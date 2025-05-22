package encoding

import (
	"fmt"
	"strings"
)

// NormalizeTxHashHex defines canonical and unambiguous representation for a
// transaction hash hexadecimal string; lower-case.
func NormalizeTxHashHex(txHash string) string {
	return strings.ToLower(txHash)
}

// TxHashBytesToNormalizedHex converts a transaction hash bytes to a normalized
// hexadecimal string representation.
func TxHashBytesToNormalizedHex(txHash []byte) string {
	return NormalizeTxHashHex(fmt.Sprintf("%x", txHash))
}

// NormalizeMorseHexAddress defines canonical and unambiguous representation for a
// morse address hexadecimal string; upper-case.
func NormalizeMorseHexAddress(morseAddress string) string {
	return strings.ToUpper(morseAddress)
}
