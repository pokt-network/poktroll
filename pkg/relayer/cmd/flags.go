package cmd

var (
	// Relay miner config file path from `--config` flag.
	flagRelayMinerConfig string
	// Cosmos node RPC URL flag value.
	flagNodeRPCURL string
	// Cosmos node GRPC URL flag value.
	flagNodeGRPCURL string
	// Cosmos node GRPC insecure flag value.
	flagNodeGRPCInsecure bool
	// Log level variable (used by cosmos and polylog).
	flagLogLevel string
	// Query caching flag value.
	flagQueryCaching bool
)
