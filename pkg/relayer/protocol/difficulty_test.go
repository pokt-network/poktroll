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
			bz:         []byte{0b11111111},
			difficulty: 0,
		},
		{
			bz:         []byte{0b01111111},
			difficulty: 1,
		},
		{
			bz:         []byte{0, 255},
			difficulty: 8,
		},
		{
			bz:         []byte{0, 0b01111111},
			difficulty: 9,
		},
		{
			bz:         []byte{0, 0b00111111},
			difficulty: 10,
		},
		{
			bz:         []byte{0, 0, 255},
			difficulty: 16,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("difficulty_%d_zero_bits", test.difficulty), func(t *testing.T) {
			var bz [32]byte
			copy(bz[:], test.bz)
			actualDifficulty := protocol.CountHashDifficultyBits(bz)
			require.Equal(t, test.difficulty, actualDifficulty)
		})
	}
}
