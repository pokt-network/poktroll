package keeper

import (
	"bytes"
	"testing"
)

func TestLeadingZeroBitsToTargetDifficultyHash(t *testing.T) {
	tests := []struct {
		name               string
		numLeadingZeroBits int
		numBytes           int
		expected           []byte
	}{
		{
			name:               "0 leading 0 bits in 1 byte",
			numLeadingZeroBits: 0,
			numBytes:           1,
			expected:           []byte{0b11111111},
		},
		{
			name:               "full zero bytes (16 leading 0 bits in 32 bytes)",
			numLeadingZeroBits: 16,
			numBytes:           32,
			expected: append(
				[]byte{0b00000000, 0b00000000},
				makeBytesFullOfOnes(30)...,
			),
		},
		{
			name:               "partial byte (20 leading 0 bits in 32 bytes)",
			numLeadingZeroBits: 20,
			numBytes:           32,
			expected: append(
				[]byte{0b00000000, 0b00000000, 0b00001111},
				makeBytesFullOfOnes(29)...,
			),
		},
		{
			name:               "another partial byte (10 leading 0 bits in 32 bytes)",
			numLeadingZeroBits: 10,
			numBytes:           32,
			expected: append(
				[]byte{0b00000000, 0b00111111},
				makeBytesFullOfOnes(30)...,
			),
		},
		{
			name:               "edge case 1 bit (1 leading 0 bits in 32 bytes)",
			numLeadingZeroBits: 1,
			numBytes:           32,
			expected: append(
				[]byte{0b01111111},
				makeBytesFullOfOnes(31)...,
			),
		},
		{
			name:               "exact byte boundary (24 leading 0 bits in 32 bytes)",
			numLeadingZeroBits: 24,
			numBytes:           32,
			expected: append(
				[]byte{0b00000000, 0b00000000, 0b00000000},
				makeBytesFullOfOnes(29)...,
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := leadingZeroBitsToTargetDifficultyHash(tt.numLeadingZeroBits, tt.numBytes)
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("got %x, expected %x", result, tt.expected)
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
