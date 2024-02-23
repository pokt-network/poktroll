package protocol_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/relayer/protocol"
)

func TestCountDifficultyBits(t *testing.T) {
	tests := []struct {
		bz         []byte
		difficulty int
	}{
		{
			bz:         []byte{0b11111111, 255, 255, 255},
			difficulty: 0,
		},
		{
			bz:         []byte{0b01111111, 255, 255, 255},
			difficulty: 1,
		},
		{
			bz:         []byte{0, 255, 255, 255},
			difficulty: 8,
		},
		{
			bz:         []byte{0, 0b01111111, 255, 255},
			difficulty: 9,
		},
		{
			bz:         []byte{0, 0b00111111, 255, 255},
			difficulty: 10,
		},
		{
			bz:         []byte{0, 0, 255, 255},
			difficulty: 16,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("difficulty_%d_zero_bits", test.difficulty), func(t *testing.T) {
			actualDifficulty, err := protocol.CountDifficultyBits(test.bz)
			require.NoError(t, err)
			require.Equal(t, test.difficulty, actualDifficulty)
		})
	}
}

func TestCountDifficultyBits_Error(t *testing.T) {
	_, err := protocol.CountDifficultyBits([]byte{0, 0, 0, 0})
	require.ErrorIs(t, err, protocol.ErrDifficulty)
	require.ErrorContains(t, err, "difficulty matches bytes length")
}
