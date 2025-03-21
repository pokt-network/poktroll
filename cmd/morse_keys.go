package cmd

// ArmoredJson is a data structure which is used to (de)serialize the encrypted exported Morse private key file.
type ArmoredJson struct {
	Kdf        string `json:"kdf" yaml:"kdf"`
	Salt       string `json:"salt" yaml:"salt"`
	SecParam   string `json:"secparam" yaml:"secparam"`
	Hint       string `json:"hint" yaml:"hint"`
	Ciphertext string `json:"ciphertext" yaml:"ciphertext"`
}
