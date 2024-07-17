package protocol

import (
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetDifficultyFromHash(t *testing.T) {
	tests := []struct {
		desc               string
		hashHex            string
		expectedDifficulty int64
	}{
		{
			desc:               "Difficulty 1",
			hashHex:            "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			expectedDifficulty: 1,
		},
		{
			desc:               "Difficulty 2",
			hashHex:            "7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			expectedDifficulty: 2,
		},
		{
			desc:               "Difficulty 4",
			hashHex:            "3fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			expectedDifficulty: 4,
		},
		{
			desc:               "Highest difficulty",
			hashHex:            "0000000000000000000000000000000000000000000000000000000000000001",
			expectedDifficulty: new(big.Int).SetBytes(Difficulty1HashBz).Int64(),
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			hashBytes, err := hex.DecodeString(test.hashHex)
			if err != nil {
				t.Fatalf("failed to decode hash: %v", err)
			}

			var hashBz [sha256.Size]byte
			copy(hashBz[:], hashBytes)

			difficulty := GetDifficultyFromHash(hashBz)
			t.Logf("test: %s, difficulty: %d", test.desc, difficulty)
			require.Equal(t, test.expectedDifficulty, difficulty)
		})
	}
}
