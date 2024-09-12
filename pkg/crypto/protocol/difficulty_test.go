package protocol

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
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

func TestComputeNewDifficultyHash(t *testing.T) {
	tests := []struct {
		desc                        string
		numRelaysTarget             uint64
		relaysEma                   uint64
		expectedRelayDifficultyHash []byte
	}{
		{
			desc:                        "Relays Target > Relays EMA",
			numRelaysTarget:             100,
			relaysEma:                   50,
			expectedRelayDifficultyHash: defaultDifficulty(),
		},
		{
			desc:                        "Relays Target == Relays EMA",
			numRelaysTarget:             100,
			relaysEma:                   100,
			expectedRelayDifficultyHash: defaultDifficulty(),
		},
		{
			desc:            "Relays Target < Relays EMA",
			numRelaysTarget: 50,
			relaysEma:       100,
			expectedRelayDifficultyHash: append(
				[]byte{0b01111111},
				makeBytesFullOfOnes(31)...,
			),
		},
		{
			desc:            "Relays Target << Relays EMA",
			numRelaysTarget: 50,
			relaysEma:       200,
			expectedRelayDifficultyHash: append(
				[]byte{0b00111111},
				makeBytesFullOfOnes(31)...,
			),
		},
		{
			desc:            "Relays Target << Relays EMA",
			numRelaysTarget: 50,
			relaysEma:       1000,
			expectedRelayDifficultyHash: append(
				[]byte{0b00001111},
				makeBytesFullOfOnes(31)...,
			),
		},
		{
			desc:            "Relays Target << Relays EMA",
			numRelaysTarget: 50,
			relaysEma:       10000,
			expectedRelayDifficultyHash: append(
				[]byte{0b00000001},
				makeBytesFullOfOnes(31)...,
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			newRelayDifficultyTargetHash := ComputeNewDifficultyTargetHash(BaseRelayDifficultyHashBz, tt.numRelaysTarget, tt.relaysEma)

			// NB: An increase in difficulty is indicated by a decrease in the target hash
			didDifficultyIncrease := bytes.Compare(newRelayDifficultyTargetHash, tt.expectedRelayDifficultyHash) < 1
			require.True(t, didDifficultyIncrease,
				"expected difficulty.TargetHash (%x) to be less than or equal to expectedRelayMiningDifficulty.TargetHash (%x)",
				newRelayDifficultyTargetHash, tt.expectedRelayDifficultyHash,
			)
		})
	}
}

func Test_EnsureRelayMiningMultiplierIsProportional(t *testing.T) {
	// Target Num Relays is the target number of volume applicable relays a session tree should have.
	const (
		targetNumRelays   = uint64(10e2) // Target number of volume applicable relays
		lowVolumeService  = 1e4          // Number of actual off-chain relays serviced by a RelayMiner
		highVolumeService = 1e6          // Number of actual off-chain relays serviced by a RelayMiner
		allowableDelta    = 0.05         // Allow a 5% error margin between estimated probabilities and results
	)

	highVolumeSvcDifficultyHash := ComputeNewDifficultyTargetHash(BaseRelayDifficultyHashBz, targetNumRelays, highVolumeService)
	highVolumeRelayProbabilityRat := RelayDifficultyProbability(highVolumeSvcDifficultyHash)
	highVolumeRelayProbability, _ := highVolumeRelayProbabilityRat.Float64()
	highVolumeRelayMultiplierRat := RelayDifficultyMultiplier(highVolumeSvcDifficultyHash)
	highVolumeRelayMultiplier, _ := highVolumeRelayMultiplierRat.Float64()

	lowVolumeSvcDifficultyHash := ComputeNewDifficultyTargetHash(BaseRelayDifficultyHashBz, targetNumRelays, lowVolumeService)
	lowVolumeRelayProbabilityRat := RelayDifficultyProbability(lowVolumeSvcDifficultyHash)
	lowVolumeRelayProbability, _ := lowVolumeRelayProbabilityRat.Float64()
	// lowVolumeRelayMultiplier := RelayDifficultyMultiplier(lowVolumeSvcDifficultyHash)

	numApplicableHighVolumeSvcRelays := 0
	numActualHighVolumeSvcRelays := 0
	for {
		bytes := generateRandomRelayHash(t)
		if IsRelayVolumeApplicable(bytes, highVolumeSvcDifficultyHash) {
			numApplicableHighVolumeSvcRelays++
		}
		numActualHighVolumeSvcRelays++
		if numApplicableHighVolumeSvcRelays >= int(targetNumRelays) {
			break
		}
	}
	numEstimatedHighVolumeSvcRelays := float64(numApplicableHighVolumeSvcRelays) * highVolumeRelayMultiplier
	fractionHighVolumeSvcRelays := float64(numApplicableHighVolumeSvcRelays) / float64(numActualHighVolumeSvcRelays)

	numApplicableLowVolumeSvcRelays := 0
	numActualLowVolumeSvcRelays := 0
	for {
		bytes := generateRandomRelayHash(t)
		if IsRelayVolumeApplicable(bytes, lowVolumeSvcDifficultyHash) {
			numApplicableLowVolumeSvcRelays++
		}
		numActualLowVolumeSvcRelays++
		if numApplicableLowVolumeSvcRelays >= int(targetNumRelays) {
			break
		}
	}
	fractionLowVolumeSvcRelays := float64(numApplicableLowVolumeSvcRelays) / float64(numActualLowVolumeSvcRelays)

	// Ensure probabilities of a relay being applicable is within the allowable delta
	require.InDelta(t, highVolumeRelayProbability, fractionHighVolumeSvcRelays, allowableDelta*highVolumeRelayProbability)
	require.InDelta(t, lowVolumeRelayProbability, fractionLowVolumeSvcRelays, allowableDelta*lowVolumeRelayProbability)

	fmt.Println(numEstimatedHighVolumeSvcRelays, numActualHighVolumeSvcRelays)
	// fmt.Println(fractionHighVolumeSvcRelays, highVolumeRelayProbability)
	// fmt.Println(fractionLowVolumeSvcRelays, lowVolumeRelayProbability)
	// fmt.Println(numActualHighVolumeSvcRelays, numActualLowVolumeSvcRelays)
}

func Test_EnsureRelayMiningProbabilityIsProportional(t *testing.T) {
	// Target Num Relays is the target number of volume applicable relays
	// a session tree should have.
	const targetNumRelays = uint64(10e4)

	// numActualRelays aims to simulate the actual (i.e. off-chain) number of relays
	// a RelayMiner would service successfully.
	for numActualRelays := uint64(1); numActualRelays < 1e18; numActualRelays *= 10 {
		// Compute the relay mining difficulty corresponding to the actual number of relays
		// to match the target number of relays.
		targetDifficultyHash := ComputeNewDifficultyTargetHash(BaseRelayDifficultyHashBz, targetNumRelays, numActualRelays)

		// The probability that a relay is a volume applicable relay
		relayProbability := RelayDifficultyProbability(targetDifficultyHash)

		fmt.Println(ratToUint64(t, RelayDifficultyMultiplier(targetDifficultyHash)))
		r := ScaleRelays(t, relayProbability, numActualRelays)
		if numActualRelays < targetNumRelays {
			require.InDelta(t, numActualRelays, r, 2)
		} else {
			require.InDelta(t, targetNumRelays, r, 2)
		}
	}
}
func makeBytesFullOfOnes(length int) []byte {
	output := make([]byte, length)
	for i := range output {
		output[i] = 0b11111111
	}
	return output
}

func defaultDifficulty() []byte {
	return makeBytesFullOfOnes(32)
}

func generateRandomBytes(t *testing.T, n int) []byte {
	bytes := make([]byte, n)
	_, err := rand.Read(bytes)
	require.NoError(t, err)
	return bytes
}

func generateRandomRelayHash(t *testing.T) []byte {
	return generateRandomBytes(t, 32)
}
