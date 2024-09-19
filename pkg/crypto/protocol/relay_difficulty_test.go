package protocol

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"math/big"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"gotest.tools/assert"
)

func TestRelayDifficulty_GetRelayDifficultyMultiplier(t *testing.T) {
	tests := []struct {
		desc               string
		hashHex            string
		expectedDifficulty string
	}{
		{
			desc:               "Difficulty 1",
			hashHex:            "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			expectedDifficulty: "1",
		},
		{
			desc:               "Difficulty 2",
			hashHex:            "7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			expectedDifficulty: "2",
		},
		{
			desc:               "Difficulty 4",
			hashHex:            "3fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			expectedDifficulty: "4",
		},
		{
			desc:               "Highest difficulty",
			hashHex:            "0000000000000000000000000000000000000000000000000000000000000001",
			expectedDifficulty: "115792089237316195423570985008687907853269984665640564039457584007913129639935",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			hashBytes, err := hex.DecodeString(test.hashHex)
			if err != nil {
				t.Fatalf("failed to decode hash: %v", err)
			}

			// Get the difficulty multiplier as a quotient of arbitrary precision
			difficultyMultiplierRat := GetRelayDifficultyMultiplier(hashBytes)
			require.NotNil(t, difficultyMultiplierRat)

			// Extract the numerator and denominator
			numerator := difficultyMultiplierRat.Num()
			denominator := difficultyMultiplierRat.Denom()

			// Determine if it fits within an int64
			difficultyMultiplierInt, err := strconv.ParseInt(test.expectedDifficulty, 10, 64)
			if err != nil {
				require.ErrorContains(t, err, "value out of range", "the only expected error for large numbers is out of range")
				require.Equal(t, "1", denominator.String(), "denominator should be 1 when value is out of range")
				require.Equal(t, test.expectedDifficulty, numerator.String())
			} else {

				// Compute quotient and remainder
				quotient := new(big.Int)
				remainder := new(big.Int)
				quotient.DivMod(numerator, denominator, remainder)

				require.NoError(t, err)
				require.Equal(t, difficultyMultiplierInt, quotient.Int64())
			}
		})
	}
}

func TestRelayDifficulty_IsRelayVolumeApplicable(t *testing.T) {
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

func TestRelayDifficulty_ComputeNewDifficultyHash(t *testing.T) {
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
				"expected difficulty.TargetHash (%x) to be equal to expectedRelayMiningDifficulty.TargetHash (%x)",
				newRelayDifficultyTargetHash, tt.expectedRelayDifficultyHash,
			)
		})
	}
}

// TestScaleDifficultyTargetHash tests the scaling of a target hash by a given ratio.
// Some expectations are manually adjusted to account for some precision loss in the
// implementation.
func TestRelayDifficulty_ScaleDifficultyTargetHash(t *testing.T) {
	tests := []struct {
		desc                  string
		currDifficultyHashHex string
		ratio                 float64
		expectedHashHex       string
	}{
		{
			desc:                  "Scale by 0.5",
			currDifficultyHashHex: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			ratio:                 0.5,
			expectedHashHex:       "7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		{
			desc:                  "Scale by 2",
			currDifficultyHashHex: "7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			ratio:                 2,
			expectedHashHex:       "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe",
		},
		{
			desc:                  "Scale by 0.25",
			currDifficultyHashHex: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			ratio:                 0.25,
			expectedHashHex:       "3fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		{
			desc:                  "Scale by 4",
			currDifficultyHashHex: "3fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			ratio:                 4,
			expectedHashHex:       "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc",
		},
		{
			desc:                  "Scale by 1 (no change)",
			currDifficultyHashHex: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			ratio:                 1,
			expectedHashHex:       "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		{
			desc:                  "Scale by 0.1",
			currDifficultyHashHex: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			ratio:                 0.1,
			expectedHashHex:       "19999999999999ffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		{
			desc:                  "Scale by 10",
			currDifficultyHashHex: "1999999999999999999999999999999999999999999999999999999999999999",
			ratio:                 10,
			expectedHashHex:       "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8",
		},
		{
			desc:                  "Scale by 10e-12",
			currDifficultyHashHex: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			ratio:                 10e-12,
			expectedHashHex:       "000000000afebff0bcb24a7fffffffffffffffffffffffffffffffffffffffff",
		},
		{
			desc:                  "Scale by 10e12",
			currDifficultyHashHex: "000000000afebff0bcb24a7fffffffffffffffffffffffffffffffffffffffff",
			ratio:                 10e12,
			expectedHashHex:       "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		{
			desc:                  "Maxes out at BaseRelayDifficulty",
			currDifficultyHashHex: "3fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			ratio:                 10,
			expectedHashHex:       "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			currHashBz, targetErr := hex.DecodeString(test.currDifficultyHashHex)
			require.NoError(t, targetErr)

			expectedHashBz, expectedErr := hex.DecodeString(test.expectedHashHex)
			require.NoError(t, expectedErr)

			ratio := new(big.Float).SetFloat64(test.ratio)
			scaledDifficultyHash := ScaleRelayDifficultyHash(currHashBz, ratio)
			assert.Equal(t, len(scaledDifficultyHash), len(currHashBz))

			// Ensure the scaled difficulty hash equals the one provided
			require.Zero(t, bytes.Compare(expectedHashBz, scaledDifficultyHash),
				"expected difficulty hash %x, but got %x", expectedHashBz, scaledDifficultyHash)
		})
	}
}

func TestRelayDifficulty_EnsureRelayMiningProbabilityIsProportional(t *testing.T) {
	// Target Num Relays is the target number of volume applicable relays
	// a session tree should have.
	const targetNumRelays = uint64(10e4)

	// numEstimatedRelays aims to simulate the actual (i.e. off-chain) number of relays
	// a RelayMiner would service successfully.
	for numEstimatedRelays := uint64(1); numEstimatedRelays < 1e18; numEstimatedRelays *= 10 {
		// Compute the relay mining difficulty corresponding to the actual number of relays
		// to match the target number of relays.
		targetDifficultyHash := ComputeNewDifficultyTargetHash(BaseRelayDifficultyHashBz, targetNumRelays, numEstimatedRelays)

		// The probability that a relay is a volume applicable relay
		relayProbability := GetRelayDifficultyProbability(targetDifficultyHash)

		volumeApplicableRelays := scaleRelaysFromActualToTarget(t, relayProbability, numEstimatedRelays)
		if numEstimatedRelays < targetNumRelays {
			require.InDelta(t, numEstimatedRelays, volumeApplicableRelays, 2)
		} else {
			require.InDelta(t, targetNumRelays, volumeApplicableRelays, 2)
		}
	}
}

// scaleRelaysFromActualToTarget scales the number of relays (i.e. estimated offchain serviced relays)
// down to the number of expected on-chain volume applicable relays
func scaleRelaysFromActualToTarget(t *testing.T, relayDifficultyProbability *big.Rat, numRelays uint64) uint64 {
	mr := new(big.Rat).SetUint64(numRelays)
	result := new(big.Rat).Mul(relayDifficultyProbability, mr)
	num := result.Num()
	denom := result.Denom()
	quotient := new(big.Int).Div(num, denom)
	require.True(t, quotient.IsUint64(), "value out of range for uint64")
	return quotient.Uint64()
}

func TestRelayDifficulty_EnsureRelayMiningMultiplierIsProportional(t *testing.T) {
	// Target Num Relays is the target number of volume applicable relays a session tree should have.
	const (
		targetNumRelays   = uint64(10e2) // Target number of volume applicable relays
		lowVolumeService  = 1e4          // Number of actual off-chain relays serviced by a RelayMiner
		highVolumeService = 1e6          // Number of actual off-chain relays serviced by a RelayMiner
		allowableDelta    = 0.05         // Allow a 5% error margin between estimated probabilities and results
	)

	highVolumeSvcDifficultyHash := ComputeNewDifficultyTargetHash(BaseRelayDifficultyHashBz, targetNumRelays, highVolumeService)
	highVolumeRelayProbabilityRat := GetRelayDifficultyProbability(highVolumeSvcDifficultyHash)
	highVolumeRelayProbability, _ := highVolumeRelayProbabilityRat.Float64()
	highVolumeRelayMultiplierRat := GetRelayDifficultyMultiplier(highVolumeSvcDifficultyHash)
	highVolumeRelayMultiplier, _ := highVolumeRelayMultiplierRat.Float64()

	lowVolumeSvcDifficultyHash := ComputeNewDifficultyTargetHash(BaseRelayDifficultyHashBz, targetNumRelays, lowVolumeService)
	lowVolumeRelayProbabilityRat := GetRelayDifficultyProbability(lowVolumeSvcDifficultyHash)
	lowVolumeRelayProbability, _ := lowVolumeRelayProbabilityRat.Float64()
	lowVolumeRelayMultiplierRat := GetRelayDifficultyMultiplier(lowVolumeSvcDifficultyHash)
	lowVolumeRelayMultiplier, _ := lowVolumeRelayMultiplierRat.Float64()

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
	numEstimatedLowVolumeSvcRelays := float64(numApplicableLowVolumeSvcRelays) * lowVolumeRelayMultiplier
	fractionLowVolumeSvcRelays := float64(numApplicableLowVolumeSvcRelays) / float64(numActualLowVolumeSvcRelays)

	// Ensure probabilities of a relay being applicable is within the allowable delta
	require.InDelta(t, highVolumeRelayProbability, fractionHighVolumeSvcRelays, allowableDelta*highVolumeRelayProbability)
	require.InDelta(t, lowVolumeRelayProbability, fractionLowVolumeSvcRelays, allowableDelta*lowVolumeRelayProbability)

	// Ensure the estimated number of relays is within the allowable delta
	require.InDelta(t, numEstimatedHighVolumeSvcRelays, float64(numActualHighVolumeSvcRelays), allowableDelta*numEstimatedHighVolumeSvcRelays)
	require.InDelta(t, numEstimatedLowVolumeSvcRelays, float64(numActualLowVolumeSvcRelays), allowableDelta*numEstimatedLowVolumeSvcRelays)

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
