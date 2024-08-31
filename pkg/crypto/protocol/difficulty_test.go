package protocol

import (
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
			expectedDifficulty: new(big.Int).SetBytes(BaseRelayDifficultyHashBz).Int64(),
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			hashBytes, err := hex.DecodeString(test.hashHex)
			if err != nil {
				t.Fatalf("failed to decode hash: %v", err)
			}

			var hashBz [RelayHasherSize]byte
			copy(hashBz[:], hashBytes)

			difficulty := GetDifficultyFromHash(hashBz)
			t.Logf("test: %s, difficulty: %d", test.desc, difficulty)
			require.Equal(t, test.expectedDifficulty, difficulty)
		})
	}
}

func TestGetDifficultyFromHash_Incremental(t *testing.T) {
	for numRelays := 1e3; numRelays < 1e18; numRelays *= 10 {
	}
}

func TestIsRelayVolumeApplicable(t *testing.T) {
	tests := []struct {
		desc                     string
		relayHashHex             string
		targetHashHex            string
		expectedVolumeApplicable bool
	}{
		{
			desc:                     "Applicable: relayHash << targetHash",
			relayHashHex:             "000fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			targetHashHex:            "00ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			expectedVolumeApplicable: true,
		},
		{
			desc:                     "Applicable: relayHash < targetHash",
			relayHashHex:             "00efffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			targetHashHex:            "00ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			expectedVolumeApplicable: true,
		},
		{
			desc:                     "Not Applicable: relayHash = targetHash",
			relayHashHex:             "00ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			targetHashHex:            "00ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			expectedVolumeApplicable: false,
		},
		{
			desc:                     "Not applicable: relayHash > targetHash",
			relayHashHex:             "0effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			targetHashHex:            "00ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			expectedVolumeApplicable: false,
		},
		{
			desc:                     "Not applicable: relayHash >> targetHash",
			relayHashHex:             "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			targetHashHex:            "00ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			expectedVolumeApplicable: false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			relayHash, err := hex.DecodeString(test.relayHashHex)
			require.NoError(t, err)

			targetHash, err := hex.DecodeString(test.targetHashHex)
			require.NoError(t, err)

			require.Equal(t, test.expectedVolumeApplicable, IsRelayVolumeApplicable(relayHash, targetHash))
		})
	}
}
