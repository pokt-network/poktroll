// Package cmd holds CLI flag variables for the relayminer commands.
//
// - Used by subcommands to configure runtime behavior
// - Values are set via CLI flags
package cmd

var (
	// flagRelayMinerConfig is the relay miner config file path from `--config` flag.
	flagRelayMinerConfig string
	// flagNodeRPCURL is the Cosmos node RPC URL flag value.
	flagNodeRPCURL string
	// flagNodeGRPCURL is the Cosmos node GRPC URL flag value.
	flagNodeGRPCURL string
	// flagLogLevel is the log level variable (used by cosmos and polylog).
	flagLogLevel string
	// flagQueryCaching is the query caching flag value.
	flagQueryCaching bool
)
