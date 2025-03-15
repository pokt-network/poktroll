package cmd

const (
	flagPassphrase      = "passphrase"
	flagPassphraseShort = "p"
	flagPassphraseUsage = "the passphrase used to decrypt the exported Morse key file for signing; the user will be prompted if empty (UNLESS --no-passphrase is used)"

	flagNoPassphrase      = "no-passphrase"
	flagNoPassphraseUsage = "attempt to use an empty passphrase to decrypt the exported Morse key file for signing"
)
