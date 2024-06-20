package keeper

import (
	"math"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
)

func TestRandProbability(t *testing.T) {
	probability := prooftypes.DefaultProofRequestProbability
	tolerance := 0.01
	confidence := 0.99

	sampleSize := requiredSampleSize(float64(probability), tolerance, confidence)

	var numTrueSamples int

	// Sample concurrently to save time.
	wg := sync.WaitGroup{}
	wg.Add(sampleSize)
	for i := 0; i < sampleSize; i++ {
		go func() {
			rand, err := randProbability(int64(i))
			require.NoError(t, err)

			if rand < 0 || rand > 1 {
				t.Fatalf("secureRandFloat64() returned out of bounds value: %f", rand)
			}

			if rand <= probability {
				numTrueSamples++
			}
			wg.Done()
		}()
	}
	wg.Wait()

	expectedNumTrueSamples := float32(sampleSize) * probability
	expectedNumFalseSamples := float32(sampleSize) * (1 - probability)
	toleranceSamples := tolerance * float64(sampleSize)

	// Check that the number of samples for each outcome is within the expected range.
	numFalseSamples := sampleSize - numTrueSamples
	require.InDeltaf(t, expectedNumTrueSamples, numTrueSamples, toleranceSamples, "true samples")
	require.InDeltaf(t, expectedNumFalseSamples, numFalseSamples, toleranceSamples, "false samples")
}

// requiredSampleSize calculates the number of samples needed to achieve a desired confidence level
// for a given probability and error threshold.
// Arguments:
//   - probability: the estimated proportion of the population (e.g., 0.5 for 50%).
//   - errThreshold: the desired margin of error, which is the maximum acceptable difference between
//     the sample estimate and the true population proportion (e.g., 0.05 for ±5%).
//     A smaller threshold means higher precision, but it requires a larger sample size.
//   - confidence: the desired confidence level, which represents the likelihood that the true
//     population proportion falls within the margin of error around the sample estimate
//     (e.g., 0.95 for 95% confidence). A higher confidence level means more certainty,
//     but it also requires a larger sample size.
//
// The function uses the standard formula for sample size determination for estimating a proportion.
// For more details, see: https://en.wikipedia.org/wiki/Sample_size_determination#Estimation_of_a_proportion
func requiredSampleSize(probability, errThreshold, confidence float64) int {
	// Calculate the z-score corresponding to the desired confidence level.
	// The z-score represents the number of standard deviations a data point
	// is from the mean in a standard normal distribution. For a given confidence
	// level, this calculation finds the z-score such that the area under the
	// normal curve between -z and z equals the confidence level.
	// The calculation uses normInv to find the z-score for the tail probability
	// (1 - (1-confidence)/2), which represents the area in one tail.
	// The absolute value is taken to ensure a positive z-score.
	z := math.Abs(normInv(1 - (1-confidence)/2))

	// Calculate the number of trials needed
	n := (z * z * probability * (1 - probability)) / (errThreshold * errThreshold)

	return int(math.Ceil(n))
}

// normInv returns the inverse of the standard normal cumulative distribution function (CDF),
// which is also known as the quantile function. The z-score in this context is the value that
// corresponds to a given cumulative probability (p) in the standard normal distribution.
// This function approximates the inverse CDF using the inverse error function (erfinv).
// The relationship between the standard normal CDF (Φ) and the error function (erf) is given by:
// Φ(x) = 0.5 * (1 + erf(x / √2))
// Therefore, the inverse CDF can be expressed in terms of the inverse error function as:
// Φ^(-1)(p) = √2 * erfinv(2p - 1)
func normInv(p float64) float64 {
	return math.Sqrt2 * math.Erfinv(2*p-1)
}
