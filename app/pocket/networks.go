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
const (
	AlphaTestNetEstimatedBlockDuration = time.Minute
	BetaTestNetEstimatedBlockDuration  = 5 * time.Minute
	MainNetEstimatedBlockDuration      = time.Minute
)
