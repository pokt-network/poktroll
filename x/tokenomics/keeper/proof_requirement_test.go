// NB: Other tests in this package directory follow the convention of applying
// the _test suffix to the end of the package name (i.e. a different package).
// These tests exercise unexported code in this package and therefore MUST be in
// the same package as the code under test.
package keeper

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
)

func TestSecureRandProbability(t *testing.T) {
	probability := prooftypes.DefaultProofRequestProbability
	tolerance := 0.01
	confidence := 0.99

	sampleSize := requiredSampleSize(float64(probability), tolerance, confidence)

	samples := make(map[bool]int)
	for i := 0; i < sampleSize; i++ {
		rand, err := secureRandProbability()
		require.NoError(t, err)

		if rand < 0 || rand > 1 {
			t.Fatalf("secureRandFloat64() returned out of bounds value: %f", rand)
		}

		samples[rand <= probability]++
	}

	// Check that the number of samples for each outcome is within the expected range.
	for outcome, count := range samples {
		var expectedCount float32
		switch outcome {
		case true:
			expectedCount = float32(sampleSize) * probability
		case false:
			expectedCount = float32(sampleSize) * (1 - probability)
		}

		require.InDeltaf(t, expectedCount, count, tolerance*float64(sampleSize), "outcome: %t", outcome)
	}
}

// requiredSampleSize calculates the number of samples needed to achieve a desired confidence level
// for a given probability and margin of error.
// See: https://en.wikipedia.org/wiki/Sample_size_determination#Estimation_of_a_proportion
func requiredSampleSize(probability, margin, confidenceLevel float64) int {
	// Calculate the z-score for the desired confidence level
	z := math.Abs(normInv(1 - (1-confidenceLevel)/2))

	// Calculate the number of trials needed
	n := (z * z * probability * (1 - probability)) / (margin * margin)

	return int(math.Ceil(n))
}

// normInv returns the inverse of the standard normal cumulative distribution function
// This function approximates the inverse CDF (quantile function)
func normInv(p float64) float64 {
	return math.Sqrt2 * math.Erfinv(2*p-1)
}
