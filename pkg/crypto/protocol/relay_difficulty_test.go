package protocol

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"math/big"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRelayDifficulty_GetRelayDifficultyMultiplier(t *testing.T) {
	tests := []struct {
		desc               string
		relayDifficultyHex string
		expectedDifficulty string
	}{
		{
			desc:               "Difficulty 1",
			relayDifficultyHex: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", // all relays are payable
			expectedDifficulty: "1",
		},
		{
			desc:               "Difficulty 2",
			relayDifficultyHex: "7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", // almost all relays are payable
			expectedDifficulty: "2",
		},
		{
			desc:               "Difficulty 4",
			relayDifficultyHex: "3fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", // slightly less, but almost all relays are payable
			expectedDifficulty: "4",
		},
		{
			desc:               "Highest difficulty",
			relayDifficultyHex: "0000000000000000000000000000000000000000000000000000000000000001", // relays are almost always not payable
			expectedDifficulty: "115792089237316195423570985008687907853269984665640564039457584007913129639935",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			hashBytes, err := hex.DecodeString(test.relayDifficultyHex)
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
		relayHashHex             string // The hex representation of the relay hash
		relayDifficultyHashHex   string // The hex representation of the relay difficulty hash
		expectedVolumeApplicable bool
	}{
		{
			desc:                     "Applicable: relayHash << targetHash",
			relayHashHex:             "000fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			relayDifficultyHashHex:   "00ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			expectedVolumeApplicable: true,
		},
		{
			desc:                     "Applicable: relayHash < targetHash",
			relayHashHex:             "00efffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			relayDifficultyHashHex:   "00ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			expectedVolumeApplicable: true,
		},
		{
			desc:                     "Not Applicable: relayHash = targetHash",
			relayHashHex:             "00ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			relayDifficultyHashHex:   "00ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			expectedVolumeApplicable: false,
		},
		{
			desc:                     "Not applicable: relayHash > targetHash",
			relayHashHex:             "0effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			relayDifficultyHashHex:   "00ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			expectedVolumeApplicable: false,
		},
		{
			desc:                     "Not applicable: relayHash >> targetHash",
			relayHashHex:             "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			relayDifficultyHashHex:   "00ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			expectedVolumeApplicable: false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			relayHash, err := hex.DecodeString(test.relayHashHex)
			require.NoError(t, err)

			targetHash, err := hex.DecodeString(test.relayDifficultyHashHex)
			require.NoError(t, err)

			require.Equal(t, test.expectedVolumeApplicable, IsRelayVolumeApplicable(relayHash, targetHash))
		})
	}
}

func TestRelayDifficulty_ComputeNewDifficultyHash(t *testing.T) {
	tests := []struct {
		desc                        string
		numRelaysTarget             uint64 // the protocol's "target" for how many relays a session tree should have
		numRelaysEMA                uint64 // the actual number of relays (as an exponential moving average) a RelayMiner would service successfully
		expectedRelayDifficultyHash []byte
	}{
		{
			desc:                        "Relays Target > Relays EMA",
			numRelaysTarget:             100,
			numRelaysEMA:                50,
			expectedRelayDifficultyHash: defaultDifficulty(),
		},
		{
			desc:                        "Relays Target == Relays EMA",
			numRelaysTarget:             100,
			numRelaysEMA:                100,
			expectedRelayDifficultyHash: defaultDifficulty(),
		},
		{
			desc:            "Relays Target < Relays EMA",
			numRelaysTarget: 50,
			numRelaysEMA:    100,
			expectedRelayDifficultyHash: append(
				[]byte{0b01111111},
				makeBytesFullOfOnes(31)...,
			),
		},
		{
			desc:            "Relays Target << Relays EMA",
			numRelaysTarget: 50,
			numRelaysEMA:    800,
			expectedRelayDifficultyHash: append(
				[]byte{0b00001111},
				makeBytesFullOfOnes(31)...,
			),
		},
		{
			desc:            "Relays Target <<< Relays EMA",
			numRelaysTarget: 50,
			numRelaysEMA:    6400,
			expectedRelayDifficultyHash: append(
				[]byte{0b00000001},
				makeBytesFullOfOnes(31)...,
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			// Explanation: An increase in difficulty is indicated by a decrease in the target hash.
			// We expect the new difficulty (newRelayDifficultyHash) to be less than or equal to the base difficulty (BaseRelayDifficultyHashBz).
			// This is because numRelaysEMA is greater than numRelaysTarget.
			newRelayDifficultyTargetHash := ComputeNewDifficultyTargetHash(BaseRelayDifficultyHashBz, tt.numRelaysTarget, tt.numRelaysEMA)

			// DEV_NOTE: The number were set up to ensure the bytes equal, but we could have used LessThanOrEqualTo here
			didDifficultyIncrease := bytes.Equal(newRelayDifficultyTargetHash, tt.expectedRelayDifficultyHash)
			require.True(t, didDifficultyIncrease,
				"newDifficulty.TargetHash(%x) != expectedRelayMiningDifficulty.TargetHash(%x)",
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
		prevDifficultyHashHex string
		scalingRatio          big.Rat

		expectedScaledDifficultyHashHex string // scaled but unbounded
		expectedNewDifficultyHashHex    string // uses the scaled result but bounded
	}{
		{
			desc:                  "Scale by 1 (same number of relays)",
			prevDifficultyHashHex: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			scalingRatio:          *big.NewRat(1, 1),

			// Scaled hash == expected hash
			expectedScaledDifficultyHashHex: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			expectedNewDifficultyHashHex:    "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},

		// Scale down and up by half
		{
			desc:                  "Scale by 0.5 (allow less relays)",
			prevDifficultyHashHex: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			scalingRatio:          *big.NewRat(1, 2),

			// Scaled hash == expected hash
			expectedScaledDifficultyHashHex: "7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			expectedNewDifficultyHashHex:    "7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		{
			desc:                  "Scale by 2 (allow more relays)",
			prevDifficultyHashHex: "7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			scalingRatio:          *big.NewRat(2, 1),

			// Scaled hash == expected hash
			expectedScaledDifficultyHashHex: "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe",
			expectedNewDifficultyHashHex:    "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe",
		},

		// Scale down and up by 4
		{
			desc:                  "Scale by 0.25 (allow less relays)",
			prevDifficultyHashHex: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			scalingRatio:          *big.NewRat(1, 4),

			// Scaled hash == expected hash
			expectedScaledDifficultyHashHex: "3fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			expectedNewDifficultyHashHex:    "3fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
		{
			desc:                  "Scale by 4 (allow more relays)",
			prevDifficultyHashHex: "3fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			scalingRatio:          *big.NewRat(4, 1),

			// Scaled hash == expected hash
			expectedScaledDifficultyHashHex: "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc",
			expectedNewDifficultyHashHex:    "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc",
		},

		// Scale down and up by 10
		{
			desc:                  "Scale by 0.1 (allow less relays)",
			prevDifficultyHashHex: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			scalingRatio:          *big.NewRat(1, 10),

			// Scaled hash == expected hash
			// Scaling down 0xff..ff by 10 leads a non-Int result (...987.5 rounded down to ...987),
			// making a scaling up of the result (0x19..99) by 10 not equal to the original hash.
			expectedScaledDifficultyHashHex: "1999999999999999999999999999999999999999999999999999999999999999",
			expectedNewDifficultyHashHex:    "1999999999999999999999999999999999999999999999999999999999999999",
		},
		{
			desc:                  "Scale by 10 (allow more relays)",
			prevDifficultyHashHex: "1999999999999999999999999999999999999999999999999999999999999999",
			scalingRatio:          *big.NewRat(10, 1),

			// Scaled hash == expected hash
			expectedScaledDifficultyHashHex: "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffa",
			expectedNewDifficultyHashHex:    "fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffa",
		},

		// Scale down and up by 10e-12 and 10e12
		{
			desc:                  "Scale by 10e-12 (allow less relays)",
			prevDifficultyHashHex: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			scalingRatio:          *big.NewRat(1, 10e12),

			// Scaled hash != expected hash
			expectedScaledDifficultyHashHex: "1c25c268497681c2650cb4be40d60df7311e9872477f201c409ec0",
			expectedNewDifficultyHashHex:    "00000000001c25c268497681c2650cb4be40d60df7311e9872477f201c409ec0",
		},
		{
			desc:                  "Scale by 10e12 (allow more relays)",
			prevDifficultyHashHex: "00000000001c25c268497681c2650cb4be40d60df7311e9872477f201c409ec0",
			scalingRatio:          *big.NewRat(10e12, 1),

			// Scaled hash == expected hash
			expectedScaledDifficultyHashHex: "fffffffffffffffffffffffffffffffffffffffffffffffffffff8cd94b80000",
			expectedNewDifficultyHashHex:    "fffffffffffffffffffffffffffffffffffffffffffffffffffff8cd94b80000",
		},
		{
			desc:                  "Scale by 10e-12 (allow more relays) padding",
			prevDifficultyHashHex: "0fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			scalingRatio:          *big.NewRat(1, 10e12),

			// Scaled hash != expected hash: Padding
			expectedScaledDifficultyHashHex: "01c25c268497681c2650cb4be40d60df7311e9872477f201c409ec",
			expectedNewDifficultyHashHex:    "000000000001c25c268497681c2650cb4be40d60df7311e9872477f201c409ec",
		},
		{
			desc:                  "Scale by 10e12 (allow more relays) truncating",
			prevDifficultyHashHex: "0000000fffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			scalingRatio:          *big.NewRat(10e12, 1),

			// Scaled hash != expected hash: Truncating
			expectedScaledDifficultyHashHex: "9184e729fffffffffffffffffffffffffffffffffffffffffffffffff6e7b18d6000",
			expectedNewDifficultyHashHex:    "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			currHashBz, err := hex.DecodeString(test.prevDifficultyHashHex)
			require.NoError(t, err)

			// Verify the scaled difficulty hash (raw, unbounded and not used in the protocol)
			expectedScaledHashBz, err := hex.DecodeString(test.expectedScaledDifficultyHashHex)
			require.NoError(t, err)

			scaledDifficultyHash := ScaleRelayDifficultyHash(currHashBz, &test.scalingRatio)

			isScaledHashAsExpected := bytes.Equal(expectedScaledHashBz, scaledDifficultyHash)
			require.True(t, isScaledHashAsExpected, "expected scaled (unbounded) difficulty hash %x, but got %x", expectedScaledHashBz, scaledDifficultyHash)

			// Verify the new difficulty hash (bounded, and used in the protocol)
			expectedNewHashBz, err := hex.DecodeString(test.expectedNewDifficultyHashHex)
			require.NoError(t, err)

			targetNumRelays := test.scalingRatio.Num().Uint64()
			newRelaysEma := test.scalingRatio.Denom().Uint64()

			newDifficultyHash := ComputeNewDifficultyTargetHash(currHashBz, targetNumRelays, newRelaysEma)
			isNewHashAsExpected := bytes.Equal(expectedNewHashBz, newDifficultyHash)
			require.True(t, isNewHashAsExpected, "expected new (bounded) difficulty hash %x, but got %x", expectedNewHashBz, newDifficultyHash)

			// If the scaled (raw) difficulty does not equal the new (bounded) difficulty, then one of
			// two things happened:
			// 1. We scaled the difficulty down so much (ratio < 1) and the new difficulty was padded appropriately
			// 2. We scaled the difficulty up so much (ratio > 1) and the new difficulty was truncated appropriately
			if test.expectedNewDifficultyHashHex != test.expectedScaledDifficultyHashHex {
				require.NotEqual(t, test.scalingRatio, 1, "should not reach this code path if scaling ratio is 1")
				// New difficulty was padded
				if targetNumRelays < newRelaysEma {
					require.Less(t, len(expectedScaledHashBz), len(newDifficultyHash))
					require.Equal(t, len(expectedNewHashBz), len(newDifficultyHash), "scaled down difficulty should have been padded")
				} else if targetNumRelays > newRelaysEma {
					require.Greater(t, len(expectedScaledHashBz), len(newDifficultyHash))
					require.Equal(t, len(expectedNewHashBz), len(newDifficultyHash), "scaled down difficulty should have been padded")
				}
			}
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
			require.InDelta(t, numEstimatedRelays, volumeApplicableRelays, 1)
		} else {
			require.InDelta(t, targetNumRelays, volumeApplicableRelays, 1)
		}
	}
}

// scaleRelaysFromActualToTarget scales the number of relays (i.e. estimated offchain serviced relays)
// down to the number of expected on-chain volume applicable relays
func scaleRelaysFromActualToTarget(t *testing.T, relayDifficultyProbability *big.Rat, numRelays uint64) uint64 {
	numRelaysRat := new(big.Rat).SetUint64(numRelays)
	volumeApplicableRelaysRat := new(big.Rat).Mul(relayDifficultyProbability, numRelaysRat)

	numerator := volumeApplicableRelaysRat.Num()
	denominator := volumeApplicableRelaysRat.Denom()

	numRelaysTarget := new(big.Int).Div(numerator, denominator)
	require.True(t, numRelaysTarget.IsUint64(), "value out of range for uint64")

	return numRelaysTarget.Uint64()
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
	highVolumeRelayProbabilityFloat, _ := highVolumeRelayProbabilityRat.Float64()
	highVolumeRelayMultiplierRat := GetRelayDifficultyMultiplier(highVolumeSvcDifficultyHash)
	highVolumeRelayMultiplierFloat, _ := highVolumeRelayMultiplierRat.Float64()

	lowVolumeSvcDifficultyHash := ComputeNewDifficultyTargetHash(BaseRelayDifficultyHashBz, targetNumRelays, lowVolumeService)
	lowVolumeRelayProbabilityRat := GetRelayDifficultyProbability(lowVolumeSvcDifficultyHash)
	lowVolumeRelayProbabilityFloat, _ := lowVolumeRelayProbabilityRat.Float64()
	lowVolumeRelayMultiplierRat := GetRelayDifficultyMultiplier(lowVolumeSvcDifficultyHash)
	lowVolumeRelayMultiplierFloat, _ := lowVolumeRelayMultiplierRat.Float64()

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
	numEstimatedHighVolumeSvcRelays := float64(numApplicableHighVolumeSvcRelays) * highVolumeRelayMultiplierFloat
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
	numEstimatedLowVolumeSvcRelays := float64(numApplicableLowVolumeSvcRelays) * lowVolumeRelayMultiplierFloat
	fractionLowVolumeSvcRelays := float64(numApplicableLowVolumeSvcRelays) / float64(numActualLowVolumeSvcRelays)

	// Ensure probabilities of a relay being applicable is within the allowable delta
	require.InDelta(t, highVolumeRelayProbabilityFloat, fractionHighVolumeSvcRelays, allowableDelta*highVolumeRelayProbabilityFloat)
	require.InDelta(t, lowVolumeRelayProbabilityFloat, fractionLowVolumeSvcRelays, allowableDelta*lowVolumeRelayProbabilityFloat)

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
