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

// NormalizeMorseAddress defines canonical and unambiguous representation for a
// morse address or module name; upper-case.
func NormalizeMorseAddress(morseAddress string) string {
	return strings.ToUpper(morseAddress)
}
