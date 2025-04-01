package testmigration

// TODO_TECHDEBT: This file is not part of pkg/crypto because it is intended to be removed after the migration.
// See this discussion as to why we copy-pasted and left it as is: https://github.com/pokt-network/poktroll/pull/1133/files#r2006773475

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/cometbft/cometbft/crypto"
	"github.com/cometbft/cometbft/crypto/ed25519"
	"golang.org/x/crypto/scrypt"

	"github.com/pokt-network/poktroll/cmd"
)

// DEV_NOTE: This file is necessary for E2E testing the migration module CLI.
// In particular, it is used to encrypt a Morse private key, identical to how
// the Morse CLI does it, such that the encrypted key file can be read by the
// Shannon CLI under test.

const (
	// Scrypt params (copied from Morse latest release)
	// See: https://github.com/pokt-network/pocket-core/blob/RC-0.12.0/crypto/keys/mintkey/mintkey.go
	n    = 32768
	r    = 8
	p    = 1
	klen = 32

	defaultKDF = "scrypt"
)

// BcryptSecurityParameter is a var so that it can be changed within the lcd test
// Making the bcrypt security parameter a var shouldn't be a security issue:
// One can't verify an invalid key by maliciously changing the bcrypt
// parameter during a runtime vulnerability. The main security
// threat this then exposes would be something that changes this during
// runtime before the user creates their key. This vulnerability must
// succeed to update this to that same value before every subsequent call
// to the keys command in future startups / or the attacker must get access
// to the filesystem. However, with a similar threat model (changing
// variables in runtime), one can cause the user to sign a different tx
// than what they see, which is a significantly cheaper attack then breaking
// a bcrypt hash. (Recall that the nonce still exists to break rainbow tables)
// For further notes on security parameter choice, see README.md
var BcryptSecurityParameter = 12

// NewArmoredJson returns a new ArmoredJson struct with the given fields.
func NewArmoredJson(kdf, salt, hint, ciphertext string) cmd.ArmoredJson {
	return cmd.ArmoredJson{
		Kdf:        kdf,
		Salt:       salt,
		SecParam:   strconv.Itoa(BcryptSecurityParameter),
		Hint:       hint,
		Ciphertext: ciphertext,
	}
}

// EncryptArmorPrivKey encrypts the given private key using a random salt and the
// given passphrase with the AES-256-GCM algorithm. It then encapsulates the ciphertext,
// salt, and hint in an ArmoredJson struct and returns it as a JSON-encoded string.
func EncryptArmorPrivKey(privKey ed25519.PrivKey, passphrase string, hint string) (string, error) {
	//first  encrypt the key
	saltBytes, encBytes, err := encryptPrivKey(privKey, passphrase)
	if err != nil {
		return "", err
	}

	//"armor" the encrypted key encoding it in base64
	armorStr := base64.StdEncoding.EncodeToString(encBytes)

	//create the ArmoredJson with the parameters to be able to decrypt it later.
	armoredJson := NewArmoredJson(defaultKDF, fmt.Sprintf("%X", saltBytes), hint, armorStr)

	//marshalling to json
	js, err := json.Marshal(armoredJson)
	if err != nil {
		return "", err
	}

	//return the json string
	return string(js), nil
}

// encrypt the given privKey with the passphrase using a randomly
// generated salt and the AES-256 GCM cipher. returns the salt and the
// encrypted priv key.
func encryptPrivKey(privKey ed25519.PrivKey, passphrase string) (saltBytes, encBytes []byte, _ error) {
	saltBytes = crypto.CRandBytes(16)
	key, err := scrypt.Key([]byte(passphrase), saltBytes, n, r, p, klen)
	if err != nil {
		return nil, nil, err
	}

	privKeyHexBz := hex.EncodeToString(privKey.Bytes())
	//encrypt using AES
	encBytes, err = EncryptAESGCM(key, []byte(privKeyHexBz))
	if err != nil {
		return nil, nil, err
	}

	return saltBytes, encBytes, nil
}

func EncryptAESGCM(key []byte, src []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	nonce := key[:12]
	out := gcm.Seal(nil, nonce, src, nil)
	return out, nil
}
