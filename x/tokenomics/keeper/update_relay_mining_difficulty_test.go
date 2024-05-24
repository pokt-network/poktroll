package keeper

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestComputeNewDifficultyHash(t *testing.T) {
	tests := []struct {
		desc                   string
		numRelaysTarget        uint64
		relaysEma              uint64
		expectedDifficultyHash []byte
	}{
		{
			desc:                   "Relays Target > Relays EMA",
			numRelaysTarget:        100,
			relaysEma:              50,
			expectedDifficultyHash: makeBytesFullOfOnes(32),
		},
		{
			desc:                   "Relays Target == Relays EMA",
			numRelaysTarget:        100,
			relaysEma:              100,
			expectedDifficultyHash: makeBytesFullOfOnes(32),
		},
		{
			desc:            "Relays Target < Relays EMA",
			numRelaysTarget: 50,
			relaysEma:       100,
			expectedDifficultyHash: append(
				[]byte{0b01111111},
				makeBytesFullOfOnes(31)...,
			),
		},
		{
			desc:            "Relays Target << Relays EMA",
			numRelaysTarget: 50,
			relaysEma:       200,
			expectedDifficultyHash: append(
				[]byte{0b00111111},
				makeBytesFullOfOnes(31)...,
			),
		},
		{
			desc:            "Relays Target << Relays EMA",
			numRelaysTarget: 50,
			relaysEma:       1000,
			expectedDifficultyHash: append(
				[]byte{0b00001111},
				makeBytesFullOfOnes(31)...,
			),
		},
		{
			desc:            "Relays Target << Relays EMA",
			numRelaysTarget: 50,
			relaysEma:       10000,
			expectedDifficultyHash: append(
				[]byte{0b00000001},
				makeBytesFullOfOnes(31)...,
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := computeNewDifficultyHash(tt.numRelaysTarget, tt.relaysEma)
			require.Equal(t, result, tt.expectedDifficultyHash)
		})
	}
}

func TestLeadingZeroBitsToTargetDifficultyHash(t *testing.T) {
	tests := []struct {
		desc                   string
		numLeadingZeroBits     int
		numBytes               int
		expectedDifficultyHash []byte
	}{
		{
			desc:                   "0 leading 0 bits in 1 byte",
			numLeadingZeroBits:     0,
			numBytes:               1,
			expectedDifficultyHash: []byte{0b11111111},
		},
		{
			desc:               "full zero bytes (16 leading 0 bits in 32 bytes)",
			numLeadingZeroBits: 16,
			numBytes:           32,
			expectedDifficultyHash: append(
				[]byte{0b00000000, 0b00000000},
				makeBytesFullOfOnes(30)...,
			),
		},
		{
			desc:               "partial byte (20 leading 0 bits in 32 bytes)",
			numLeadingZeroBits: 20,
			numBytes:           32,
			expectedDifficultyHash: append(
				[]byte{0b00000000, 0b00000000, 0b00001111},
				makeBytesFullOfOnes(29)...,
			),
		},
		{
			desc:               "another partial byte (10 leading 0 bits in 32 bytes)",
			numLeadingZeroBits: 10,
			numBytes:           32,
			expectedDifficultyHash: append(
				[]byte{0b00000000, 0b00111111},
				makeBytesFullOfOnes(30)...,
			),
		},
		{
			desc:               "edge case 1 bit (1 leading 0 bits in 32 bytes)",
			numLeadingZeroBits: 1,
			numBytes:           32,
			expectedDifficultyHash: append(
				[]byte{0b01111111},
				makeBytesFullOfOnes(31)...,
			),
		},
		{
			desc:               "exact byte boundary (24 leading 0 bits in 32 bytes)",
			numLeadingZeroBits: 24,
			numBytes:           32,
			expectedDifficultyHash: append(
				[]byte{0b00000000, 0b00000000, 0b00000000},
				makeBytesFullOfOnes(29)...,
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := leadingZeroBitsToTargetDifficultyHash(tt.numLeadingZeroBits, tt.numBytes)
			if !bytes.Equal(result, tt.expectedDifficultyHash) {
				t.Errorf("got %x, expected %x", result, tt.expectedDifficultyHash)
			}
		})
	}
}
func makeBytesFullOfOnes(length int) []byte {
	result := make([]byte, length)
	for i := range result {
		result[i] = 0b11111111
	}
	return result
}
