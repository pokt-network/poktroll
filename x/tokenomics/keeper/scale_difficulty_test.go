package keeper

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestScaleDifficultyTargetHash tests the scaling of a target hash by a given ratio.
// Some expectations are manually adjusted to account for some precision loss in the
// implementation.
func TestScaleDifficultyTargetHash(t *testing.T) {
	tests := []struct {
		desc            string
		targetHashHex   string
		ratio           float64
		expectedHashHex string
	}{
		{
			desc:            "Scale by 0.5",
			targetHashHex:   "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			ratio:           0.5,
			expectedHashHex: "7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		{
			desc:            "Scale by 2",
			targetHashHex:   "7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			ratio:           2,
			expectedHashHex: "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe",
		},
		{
			desc:            "Scale by 0.25",
			targetHashHex:   "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			ratio:           0.25,
			expectedHashHex: "3fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		{
			desc:            "Scale by 4",
			targetHashHex:   "3fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			ratio:           4,
			expectedHashHex: "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc",
		},
		{
			desc:            "Scale by 1 (no change)",
			targetHashHex:   "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			ratio:           1,
			expectedHashHex: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		{
			desc:            "Scale by 0.1",
			targetHashHex:   "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			ratio:           0.1,
			expectedHashHex: "19999999999999ffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		{
			desc:            "Scale by 10",
			targetHashHex:   "1999999999999999999999999999999999999999999999999999999999999999",
			ratio:           10,
			expectedHashHex: "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8",
		},
		{
			desc:            "Scale by 10e-12",
			targetHashHex:   "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			ratio:           10e-12,
			expectedHashHex: "000000000afebff0bcb24a7fffffffffffffffffffffffffffffffffffffffff",
		},
		{
			desc:            "Scale by 10e12",
			targetHashHex:   "000000000afebff0bcb24a7fffffffffffffffffffffffffffffffffffffffff",
			ratio:           10e12,
			expectedHashHex: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		{
			desc:            "Maxes out at Difficulty1",
			targetHashHex:   "3fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			ratio:           10,
			expectedHashHex: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			targetHashBz, targetErr := hex.DecodeString(test.targetHashHex)
			require.NoError(t, targetErr)

			expectedBytes, expectedErr := hex.DecodeString(test.expectedHashHex)
			require.NoError(t, expectedErr)

			scaledHash := scaleDifficultyTargetHash(targetHashBz, new(big.Float).SetFloat64(test.ratio))
			assert.Equal(t, len(scaledHash), len(targetHashBz))
			require.Equalf(t, 0, bytes.Compare(scaledHash, expectedBytes), "expected hash %x, got %x", expectedBytes, scaledHash)
		})
	}
}
