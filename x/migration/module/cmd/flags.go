// Package cmd holds CLI flag variables for the migration module.
//
// - Used by subcommands to configure runtime behavior
// - Values are set via CLI flags
package cmd

var (
	// flagInputFilePath is the path to the input file.
	flagInputFilePath string

	// flagOutputFilePath is the path to the output file.
	flagOutputFilePath string

	// flagUnsafe is a flag that enables unsafe operations. This flag must be switched on along with all unsafe operation-specific options.
	flagUnsafe bool

	// flagUnarmoredJSON is a flag that exports unarmored hex privkey. Requires --unsafe.
	flagUnarmoredJSON bool

	// flagDryRunClaim is a flag that enables dry-run mode for the claim operation.
	flagDryRunClaim bool
)

const (
	FlagUnarmoredJSON     = "unarmored-json"
	FlagUnarmoredJSONDesc = "Export unarmored hex privkey. Requires --unsafe."

	FlagUnsafe     = "unsafe"
	FlagUnsafeDesc = "Enable unsafe operations. This flag must be switched on along with all unsafe operation-specific options."

	FlagOutputFile     = "output-file"
	FlagOutputFileDesc = "Path to a file where the migration result will be written."

	FlagDryRunClaim     = "dry-run-claim"
	FlagDryRunClaimDesc = "If true, the claim transaction will be simulated (i.e. a dry run) but not broadcasted onchain."
)
