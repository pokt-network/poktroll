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

func txHashBytesToNormalizedHex(txHash []byte) string {
	return normalizeTxHashHex(fmt.Sprintf("%x", txHash))
}
