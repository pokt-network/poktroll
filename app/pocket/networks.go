package pocket

import (
	"time"
)

const (
	LocalNetChainId     = "pocket"
	AlphaTestNetChainId = "pocket-alpha"
	BetaTestNetChainId  = "pocket-beta"
	MainNetChainId      = "pocket"
)

// EstimatedBlockDurationByChainId maps chain IDs to their estimated block durations.
//
// - These estimates are derived from the consensus configuration for each network.
// - Block durations are inferred by averaging the time between recent consecutive blocks.
// - Estimations consider factors such as:
//   - network latency
//   - timeout_commit configuration
//   - Consult the consensus timeout configurations for further details:
//     https://docs.cometbft.com/v0.38/core/configuration#consensus-timeouts-explained
//
// | Network       | timeout_commit | Consensus Config                                                                                          |
// |---------------|----------------|-----------------------------------------------------------------------------------------------------------|
// | Alpha TestNet | 1m0s           | https://github.com/pokt-network/pocket-network-genesis/blob/master/shannon/testnet-alpha/config.toml#L426 |
// | Beta TestNet  | 5m0s           | https://github.com/pokt-network/pocket-network-genesis/blob/master/shannon/testnet-beta/config.toml#L426  |
// | MainNet       | 1m0s           | https://github.com/pokt-network/pocket-network-genesis/blob/master/shannon/mainnet/config.toml#L426       |
var EstimatedBlockDurationByChainId = map[string]time.Duration{
	AlphaTestNetChainId: AlphaTestNetEstimatedBlockDuration,
	BetaTestNetChainId:  BetaTestNetEstimatedBlockDuration,
	MainNetChainId:      MainNetEstimatedBlockDuration,
}

const (
	AlphaTestNetEstimatedBlockDuration = time.Minute
	BetaTestNetEstimatedBlockDuration  = 5 * time.Minute
	MainNetEstimatedBlockDuration      = time.Minute
)

const (
	LocalNetRPCURL     = "http://localhost:26657"
	AlphaTestNetRPCURL = "https://shannon-testnet-grove-rpc.alpha.poktroll.com"
	BetaTestNetRPCURL  = "https://shannon-testnet-grove-rpc.beta.poktroll.com"
	MainNetRPCURL      = "https://shannon-grove-rpc.mainnet.poktroll.com"

	LocalNetGRPCAddr = "localhost:9090"
	AlphaNetGRPCAddr = "shannon-testnet-grove-grpc.alpha.poktroll.com:443"
	BetaNetGRPCAddr  = "shannon-testnet-grove-grpc.beta.poktroll.com:443"
	MainNetGRPCAddr  = "shannon-grove-grpc.mainnet.poktroll.com:443"

	LocalNetFaucetBaseURL     = "http://localhost:8080"
	AlphaTestNetFaucetBaseURL = "shannon-testnet-grove-grpc.alpha.poktroll.com:443"
	BetaTestNetFaucetBaseURL  = "shannon-testnet-grove-grpc.beta.poktroll.com:443"
	MainNetFaucetBaseURL      = "shannon-grove-grpc.mainnet.poktroll.com:443"
)
