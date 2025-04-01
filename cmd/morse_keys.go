package cmd

// TODO_TECHDEBT: This file is not part of pkg/crypto because it is intended to be removed after the migration.

// ArmoredJson is a data structure which is used to (de)serialize the encrypted exported Morse private key file.
// Copy-pasted from https://github.com/pokt-network/pocket-core/blob/2cd25e82095dc52939fef58e9cc3deb1923c01f7/crypto/keys/mintkey/mintkey.go#L96
type ArmoredJson struct {
	Kdf        string `json:"kdf" yaml:"kdf"`
	Salt       string `json:"salt" yaml:"salt"`
	SecParam   string `json:"secparam" yaml:"secparam"`
	Hint       string `json:"hint" yaml:"hint"`
	Ciphertext string `json:"ciphertext" yaml:"ciphertext"`
}
