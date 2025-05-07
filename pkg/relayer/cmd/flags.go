package cmd

var (
	// Relay miner config file path from `--config` flag.
	flagRelayMinerConfig string
	// Cosmos node RPC URL flag value.
	flagNodeRPCURL string
	// Cosmos node GRPC URL flag value.
	flagNodeGRPCURL string
	// Log level variable (used by cosmos and polylog).
	flagLogLevel string
	// Query caching flag value.
	flagQueryCaching bool
)
