package rand_test

import (
	"encoding/binary"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	poktrand "github.com/pokt-network/poktroll/pkg/crypto/rand"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
)

func TestSeededFloat32(t *testing.T) {
	probability := prooftypes.DefaultProofRequestProbability
	tolerance := 0.01
	confidence := 0.99

	sampleSize := poktrand.RequiredSampleSize(float64(probability), tolerance, confidence)

	var numTrueSamples atomic.Int64

	// Sample concurrently to save time.
	wg := sync.WaitGroup{}
	for idx := int64(0); idx < sampleSize; idx++ {
		wg.Add(1)
		go func() {
			idxBz := make([]byte, binary.MaxVarintLen64)
			binary.PutVarint(idxBz, idx)
			randFloat, err := poktrand.SeededFloat32(idxBz)
			require.NoError(t, err)

			if randFloat < 0 || randFloat > 1 {
				t.Fatalf("secureRandFloat64() returned out of bounds value: %f", randFloat)
			}

			if randFloat <= probability {
				numTrueSamples.Add(1)
			}
			wg.Done()
		}()
	}
	wg.Wait()

	expectedNumTrueSamples := float32(sampleSize) * probability
	expectedNumFalseSamples := float32(sampleSize) * (1 - probability)
	toleranceSamples := tolerance * float64(sampleSize)

	// Check that the number of samples for each outcome is within the expected range.
	numFalseSamples := sampleSize - numTrueSamples.Load()
	require.InDeltaf(t, expectedNumTrueSamples, numTrueSamples.Load(), toleranceSamples, "true samples")
	require.InDeltaf(t, expectedNumFalseSamples, numFalseSamples, toleranceSamples, "false samples")
}
