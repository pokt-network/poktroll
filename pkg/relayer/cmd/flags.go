// Package cmd holds CLI flag variables for the relayminer commands.
//
// - Used by subcommands to configure runtime behavior
// - Values are set via CLI flags
package cmd

var (
	// flagRelayMinerConfig is the relay miner config file path from `--config` flag.
	flagRelayMinerConfig string
	// flagQueryCaching is the query caching flag value.
	flagQueryCaching bool
)
