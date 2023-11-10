package protocol

import (
	"encoding/hex"
	"strings"
)

// TODO_BLOCKER: Revisit this part of the algorithm after initial TestNet Launch.
// TODO_TEST: Add extensive tests for the core relay mining business logic.
// BytesDifficultyGreaterThan determines if the bytes exceed a certain difficulty, and it
// is used to determine if a relay is volume applicable. See the spec for more details: https://github.com/pokt-network/pocket-network-protocol
func BytesDifficultyGreaterThan(bz []byte, compDifficultyBytes int) bool {
	hexZerosPrefix := strings.Repeat("0", compDifficultyBytes*2) // 2 hex chars per byte.
	hexBz := hex.EncodeToString(bz)

	return strings.HasPrefix(hexBz, hexZerosPrefix)
}
