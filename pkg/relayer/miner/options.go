package miner

import "github.com/pokt-network/poktroll/pkg/relayer"

// WithRelayDifficultyTargetHash sets the relayDifficultyTargetHash of the miner.
func WithRelayDifficultyTargetHash(targetHash []byte) relayer.MinerOption {
	return func(mnr relayer.Miner) {
		mnr.(*miner).relayDifficultyTargetHash = targetHash
	}
}
