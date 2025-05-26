package flags

const (
	// OmittedDefaultFlagValue is used whenever a flag is required but no reasonable default value can be provided.
	// In most cases, this forces the user to specify the flag value to avoid unintended behavior.
	OmittedDefaultFlagValue = "intentionally omitting default"

	FlagLogLevel      = "log-level"
	FlagLogLevelUsage = "The logging level (debug|info|warn|error)"
	DefaultLogLevel   = "info"

	FlagLogOutput      = "log-output"
	FlagLogOutputUsage = "The logging output (file path); defaults to stdout"
	DefaultLogOutput   = "-"

	FlagPassphrase      = "passphrase"
	FlagPassphraseShort = "p"
	FlagPassphraseUsage = "the passphrase used to decrypt the exported Morse key file for signing; the user will be prompted if empty (UNLESS --no-passphrase is used)"

	FlagNoPassphrase      = "no-passphrase"
	FlagNoPassphraseUsage = "attempt to use an empty passphrase to decrypt the exported Morse key file for signing"

	FlagInputFile      = "input-file"
	FlagInputFileShort = "" // TODO_TECHDEBT: Add short flag
	FlagInputFileUsage = "An absolute or relative path to an input file that can be used to read data from. This will not be overwritten."

	FlagOutputFile      = "output-file"
	FlagOutputFileShort = "" // TODO_TECHDEBT: Add short flag
	FlagOutputFileUsage = "An absolute or relative path to an output file that can be used to write data to. Caution that this file may be updated or overwritten if it already exists."
)
