package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// A zero EMA must not panic (big.NewRat divides by it); the difficulty is kept
// unchanged. This path runs in the tokenomics EndBlocker, so a panic halts the chain.
func TestRelayDifficulty_ComputeNewDifficultyHash_ZeroEMA(t *testing.T) {
	prev := defaultDifficulty()
	var result []byte
	require.NotPanics(t, func() {
		result = ComputeNewDifficultyTargetHash(prev, 100, 0)
	})
	require.Equal(t, prev, result)
}
