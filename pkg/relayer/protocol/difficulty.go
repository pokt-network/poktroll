package protocol

import (
	"encoding/hex"
	"strings"
)

func BytesDifficultyGreaterThan(bz []byte, compDifficultyBytes int) bool {
	hexZerosPrefix := strings.Repeat("0", compDifficultyBytes*2) // 2 hex chars per byte.
	hexBz := hex.EncodeToString(bz)

	return strings.HasPrefix(hexBz, hexZerosPrefix)
}
