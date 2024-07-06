package miner

import "github.com/pokt-network/poktroll/pkg/relayer"

// WithDifficulty sets the difficulty of the miner, where difficultyBytes is the
// minimum number of leading zero bytes.
func WithDifficulty(difficultyBits uint64) relayer.MinerOption {
	return func(mnr relayer.Miner) {
		mnr.(*miner).relayDifficultyBits = difficultyBits
	}
}
