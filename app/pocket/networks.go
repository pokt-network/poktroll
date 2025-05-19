package pocket

import (
	"time"
)

const (
	AlphaTestNetChainId = "pocket-alpha"
	BetaTestNetChainId  = "pocket-beta"
	MainNetChainId      = "pocket"
)

// TODO_IN_THIS_COMMIT: comment...
var EstimatedBlockDurationByChainId = map[string]time.Duration{
	AlphaTestNetChainId: AlphaTestNetEstimatedBlockDuration,
	BetaTestNetChainId:  BetaTestNetEstimatedBlockDuration,
	MainNetChainId:      MainNetEstimatedBlockDuration,
}

// TODO_IN_THIS_COMMIT: comment...
// ... specified in the config.toml; i.e. per validator, offchain ...
// ... times can be / were inferred by averaging the duration between recent consecutive blocks...
// ... est_avg_block_time ≈ timeout_commit + 1s...
// ... actual block time depends on dynamic factors like network latency, consensus outcomes, etc...
// See: https://docs.cometbft.com/v0.38/core/configuration#consensus-timeouts-explained
/*
| Network       | timeout_commit | Consensus Config                                                                                          |
|---------------|----------------|-----------------------------------------------------------------------------------------------------------|
| Alpha TestNet | 1m0s           | https://github.com/pokt-network/pocket-network-genesis/blob/master/shannon/testnet-alpha/config.toml#L426 |
| Beta TestNet  | 5m0s           | https://github.com/pokt-network/pocket-network-genesis/blob/master/shannon/testnet-beta/config.toml#L426  |
| MainNet       | 1m0s           | https://github.com/pokt-network/pocket-network-genesis/blob/master/shannon/mainnet/config.toml#L426       |
*/
const (
	AlphaTestNetEstimatedBlockDuration = time.Minute
	BetaTestNetEstimatedBlockDuration  = 5 * time.Minute
	MainNetEstimatedBlockDuration      = time.Minute
)
