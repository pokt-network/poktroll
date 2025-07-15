// Package cmd holds CLI flag variables for the relayminer commands.
//
// - Used by subcommands to configure runtime behavior
// - Values are set via CLI flags
package cmd

var (
	// relayMinerConfigPath is the relay miner config file path from `--config` flag.
	relayMinerConfigPath string
	// flagQueryCaching is the query caching flag value.
	flagQueryCaching bool
)

const (
	FlagApp        = "app"
	FlagAppUsage   = "(Required) Staked application address"
	DefaultFlagApp = ""

	FlagPayload        = "payload"
	FlagPayloadUsage   = "(Required) JSON-RPC payload"
	DefaultFlagPayload = ""

	FlagSupplier        = "supplier"
	FlagSupplierUsage   = "(Optional) Staked Supplier address"
	DefaultFlagSupplier = ""

	FlagSupplierPublicEndpointOverride        = "supplier-public-endpoint-override"
	FlagSupplierPublicEndpointOverrideUsage   = "(Optional) Override the publicly exposed endpoint of the Supplier (useful for LocalNet testing)"
	DefaultFlagSupplierPublicEndpointOverride = ""

	FlagConfig        = "config"
	FlagConfigUsage   = "(Required) The path to the relayminer config file"
	DefaultFlagConfig = ""

	FlagCount        = "count"
	FlagCountUsage   = "(Optional) Number of requests to send (default: 1)"
	DefaultFlagCount = 1
)
