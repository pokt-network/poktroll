package tests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestComputeTotalRequests(t *testing.T) {
	tests := []struct {
		initialRelaysPerSec int64
		relaysPerSecInc     int64
		numBlocksInc        int64
		maxRelaysPerSec     int64
		expectedTotalRelays int64
	}{
		{
			initialRelaysPerSec: 1,
			relaysPerSecInc:     1,
			numBlocksInc:        1,
			maxRelaysPerSec:     5,
			expectedTotalRelays: 15,
			// NB: assumes 1 block per second
			// 1: 1 * 1
			// 2: +2 = 3
			// 3: +3 = 6
			// 4: +4 = 10
			// 5: +5 = 15
		},
		{
			initialRelaysPerSec: 10,
			relaysPerSecInc:     10,
			numBlocksInc:        3,
			maxRelaysPerSec:     100,
			expectedTotalRelays: 1650,
			// NB: assumes 1 block per second
			// 1: 10 * 3 = 30
			// 2: +20 * 3 = 90
			// 3: +30 * 3 = 180
			// 4: +40 * 3 = 300
			// 5: +50 * 3 = 450
			// 6: +60 * 3 = 630
			// 7: +70 * 3 = 840
			// 8: +80 * 3 = 1080
			// 9: +90 * 3 = 1350
			// 10: +100 * 3 = 1650
		},
	}

	for _, test := range tests {
		actual := computeTotalRequests(
			test.initialRelaysPerSec,
			test.relaysPerSecInc,
			test.numBlocksInc,
			test.maxRelaysPerSec,
		)
		require.Equal(t, test.expectedTotalRelays, int64(actual))
	}
}
