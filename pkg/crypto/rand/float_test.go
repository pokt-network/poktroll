package rand_test

import (
	"encoding/binary"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	poktrand "github.com/pokt-network/poktroll/pkg/crypto/rand"
	"github.com/pokt-network/poktroll/proto/types/proof"
)

func TestSeededFloat32(t *testing.T) {
	probability := proof.DefaultProofRequestProbability
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
			randFloat, err := poktrand.SeededFloat32(idxBz)
			require.NoError(t, err)

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

	expectedNumTrueSamples := float32(sampleSize) * probability
	expectedNumFalseSamples := float32(sampleSize) * (1 - probability)
	toleranceSamples := tolerance * float64(sampleSize)

	// Check that the number of samples for each outcome is within the expected range.
	numFalseSamples := sampleSize - numTrueSamples.Load()
	require.InDeltaf(t, expectedNumTrueSamples, numTrueSamples.Load(), toleranceSamples, "true samples")
	require.InDeltaf(t, expectedNumFalseSamples, numFalseSamples, toleranceSamples, "false samples")
}
