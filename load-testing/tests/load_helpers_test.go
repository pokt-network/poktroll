package tests

import (
	"testing"
)

func TestComputeTotalRequests(t *testing.T) {
	_ = []struct {
		initialRelaysRate   int64
		relayRateInc        int64
		relayBlocksInc      int64
		maxRelayRate        int64
		expectedTotalRelays int64
	}{
		{
			initialRelaysRate:   1,
			relayRateInc:        1,
			relayBlocksInc:      1,
			maxRelayRate:        5,
			expectedTotalRelays: 15,
			// NB: assumes 1 block per second
			// 1: 1 * 1
			// 2: +2 = 3
			// 3: +3 = 6
			// 4: +4 = 10
			// 5: +5 = 15
		},
		{
			initialRelaysRate:   5,
			relayRateInc:        5,
			relayBlocksInc:      4,
			maxRelayRate:        30,
			expectedTotalRelays: 420, // 🌲😎
			// NB: assumes 1 block per second
			// 1: 5 * 4 = 20
			// 2: +10 * 4 = 60
			// 3: +15 * 4 = 120
			// 4: +20 * 4 = 200
			// 5: +25 * 4 = 300
			// 6: +30 * 4 = 420
		},
		{
			initialRelaysRate:   10,
			relayRateInc:        10,
			relayBlocksInc:      3,
			maxRelayRate:        100,
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

	// for _, test := range tests {
	// 	actual := computeTotalRequests(
	// 		test.initialRelaysRate,
	// 		test.relayRateInc,
	// 		test.relayBlocksInc,
	// 		test.maxRelayRate,
	// 	)
	// 	require.Equal(t, test.expectedTotalRelays, int64(actual))
	// }
}
