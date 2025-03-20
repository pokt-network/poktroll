package flags

const (
	DefaultLogOutput        = "-"
	OmittedDefaultFlagValue = "explicitly omitting default"

	FlagLogLevel      = "log-level"
	FlagLogLevelUsage = "The logging level (debug|info|warn|error)"

	FlagLogOutput      = "log-output"
	FlagLogOutputUsage = "The logging output (file path); defaults to stdout"

	FlagPassphrase      = "passphrase"
	FlagPassphraseShort = "p"
	FlagPassphraseUsage = "the passphrase used to decrypt the exported Morse key file for signing; the user will be prompted if empty (UNLESS --no-passphrase is used)"

	FlagNoPassphrase      = "no-passphrase"
	FlagNoPassphraseUsage = "attempt to use an empty passphrase to decrypt the exported Morse key file for signing"
)
