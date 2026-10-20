package rand_test

import (
	"encoding/binary"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	poktrand "github.com/pokt-network/poktroll/pkg/crypto/rand"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
)

func TestSeededFloat64(t *testing.T) {
	probability := prooftypes.DefaultProofRequestProbability
	tolerance := 0.01
	confidence := 0.99

	sampleSize := poktrand.RequiredSampleSize(float64(probability), tolerance, confidence)

	var numTrueSamples atomic.Int64

	// Sample concurrently to save time.
	wg := sync.WaitGroup{}
	errCh := make(chan error, 1)
	for idx := int64(0); idx < sampleSize; idx++ {
		wg.Add(1)
		go func(idx int64) {
			idxBz := make([]byte, binary.MaxVarintLen64)
			binary.PutVarint(idxBz, idx)
			randFloat := poktrand.SeededFloat64(idxBz)

			if randFloat < 0 || randFloat > 1 {
				errCh <- fmt.Errorf("secureRandFloat64() returned out of bounds value: %f", randFloat)
				wg.Done()
				return
			}

			if randFloat <= probability {
				numTrueSamples.Add(1)
			}
			wg.Done()
		}(idx)
	}

	go func() {
		wg.Wait()
		close(errCh)
	}()

	err := <-errCh
	require.NoError(t, err)

	expectedNumTrueSamples := float64(sampleSize) * probability
	expectedNumFalseSamples := float64(sampleSize) * (1 - probability)
	toleranceSamples := tolerance * float64(sampleSize)

	// Check that the number of samples for each outcome is within the expected range.
	numFalseSamples := sampleSize - numTrueSamples.Load()
	require.InDeltaf(t, expectedNumTrueSamples, numTrueSamples.Load(), toleranceSamples, "true samples")
	require.InDeltaf(t, expectedNumFalseSamples, numFalseSamples, toleranceSamples, "false samples")
}

// TestSeededFloat64_LowProbability checks the sub-percent tail is sampled at its
// nominal rate (mainnet proof_request_probability is 0.001).
func TestSeededFloat64_LowProbability(t *testing.T) {
	const (
		probability = 0.001
		numSamples  = 2_000_000
	)

	numTrueSamples := 0
	seed := make([]byte, 8)
	for idx := uint64(0); idx < numSamples; idx++ {
		binary.BigEndian.PutUint64(seed, idx)
		if poktrand.SeededFloat64(seed) <= probability {
			numTrueSamples++
		}
	}

	// Expected 2000, stddev ~45: +/-10% is > 4 sigma.
	expected := probability * numSamples
	require.InEpsilon(t, expected, float64(numTrueSamples), 0.1)
}
