package cmd

// TODO_TECHDEBT: This file is not part of pkg/crypto because it is intended to be removed after the migration.

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/cometbft/cometbft/crypto/ed25519"
	"golang.org/x/crypto/scrypt"
	"golang.org/x/term"

	"github.com/pokt-network/poktroll/cmd"
)

// DEV_NOTE: The following code is extracted and/or derived from Morse (pocket-core) code.
// In order to avoid a direct code dependency on Morse, some code has been (minimally) duplicated.
//
// The following enables private keys which are exported by the Morse CLI to be
// deserialized and decrypted for signing in the context of claiming Morse accounts.

const (
	// Scrypt params (copied from Morse latest release)
	// See: https://github.com/pokt-network/pocket-core/blob/RC-0.12.0/crypto/keys/mintkey/mintkey.go
	n    = 32768
	r    = 8
	p    = 1
	klen = 32
)

// LoadMorsePrivateKey reads, deserializes, decrypts and returns an exported Morse private key from morseKeyExportPath.
func LoadMorsePrivateKey(morseKeyExportPath, passphrase string, noPrompt bool) (ed25519.PrivKey, error) {
	morseArmoredKeyfileBz, err := os.ReadFile(morseKeyExportPath)
	if err != nil {
		return nil, err
	}

	// Support overriding via the noPassphrase flag.
	passphrase, err = ensurePassphrase(passphrase, noPrompt)
	if err != nil {
		return nil, err
	}

	return UnarmorDecryptPrivKey(morseArmoredKeyfileBz, passphrase)
}

// ensurePassphrase returns the passphrase with surrounding whitespace removed.
// If noPrompt is false AND passphrase is empty, the user is prompted for the passphrase via stdin.
// If noPrompt is true, the user won't be prompted, even if passphrase is empty.
func ensurePassphrase(passphrase string, noPrompt bool) (string, error) {
	if passphrase == "" && noPrompt {
		return "", nil
	}

	trimmedPwd := strings.TrimSpace(passphrase)
	if trimmedPwd != "" {
		return trimmedPwd, nil
	}

	fmt.Printf("Enter Decrypt Passphrase: ")
	bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", err
	}
	fmt.Println()

	return strings.TrimSpace(string(bytePassword)), nil
}

// UnarmorDecryptPrivKey deserializes and decrypts the exported Morse private key file in armorStr using the passphrase.
func UnarmorDecryptPrivKey(armorStr []byte, passphrase string) (ed25519.PrivKey, error) {
	var privKey ed25519.PrivKey
	armoredJson := cmd.ArmoredJson{}

	// trying to unmarshal to ArmoredJson Struct
	if err := json.Unmarshal(armorStr, &armoredJson); err != nil {
		return privKey, err
	}

	// check the ArmoredJson for the correct parameters on kdf and salt
	if armoredJson.Kdf != "scrypt" {
		return privKey, fmt.Errorf("unrecognized KDF type: %s, expected scrypt", armoredJson.Kdf)
	}
	if armoredJson.Salt == "" {
		return privKey, fmt.Errorf("missing salt bytes")
	}

	// decoding the salt
	saltBytes, err := hex.DecodeString(armoredJson.Salt)
	if err != nil {
		return privKey, fmt.Errorf("error decoding salt: %w", err)
	}

	// decoding the "armored" ciphertext stored in base64
	encBytes, err := base64.StdEncoding.DecodeString(armoredJson.Ciphertext)
	if err != nil {
		return privKey, fmt.Errorf("error decoding ciphertext: %w", err)
	}

	// decrypt the actual privkey with the parameters
	privKey, err = decryptPrivKey(saltBytes, encBytes, passphrase)
	return privKey, err
}

// decryptPrivKey decrypts the exported Morse private key file (encBytes) using the given saltBytes and passphrase.
func decryptPrivKey(
	saltBytes []byte,
	encBytes []byte,
	passphrase string,
) (privKey ed25519.PrivKey, err error) {
	key, err := scrypt.Key([]byte(passphrase), saltBytes, n, r, p, klen)
	if err != nil {
		return nil, fmt.Errorf("error generating bcrypt key from passphrase: %w", err)
	}

	// decrypt using AES
	privKeyHexBz, err := decryptAESGCM(key, encBytes)
	if err != nil {
		return privKey, err
	}

	privKeyHexBz, _ = hex.DecodeString(string(privKeyHexBz))
	pk := ed25519.PrivKey(privKeyHexBz)

	return pk, err
}

// decryptAESGCM decrypts encBytes using the given key bytes.
func decryptAESGCM(key []byte, encBytes []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("unable to create a new AES cipher block: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("unable to create a new GCM cipher: %w", err)
	}

	nonce := key[:12]
	result, err := gcm.Open(nil, nonce, encBytes, nil)
	if err != nil {
		return nil, fmt.Errorf("can't Decrypt Using AES: %w", err)
	}

	return result, nil
}
