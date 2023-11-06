package tx

import (
	"fmt"
	"strings"
)

// normalizeTxHashHex defines canonical and unambiguous representation for a
// transaction hash hexadecimal string; lower-case.
func normalizeTxHashHex(txHash string) string {
	return strings.ToLower(txHash)
}

// txHashBytesToNormalizedHex converts a transaction hash bytes to a normalized
// hexadecimal string representation.
func txHashBytesToNormalizedHex(txHash []byte) string {
	return normalizeTxHashHex(fmt.Sprintf("%x", txHash))
}
