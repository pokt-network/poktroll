package testmigration

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
)

// TODO_IN_THIS_COMMIT: DEV_NOTE: this stuff is necessary for E2E testing...

const (
	// Scrypt params (copied from Morse latest release)
	// See: https://github.com/pokt-network/pocket-core/blob/RC-0.12.0/crypto/keys/mintkey/mintkey.go
	n    = 32768
	r    = 8
	p    = 1
	klen = 32

	defaultKDF = "scrypt"
)

// Make bcrypt security parameter var, so it can be changed within the lcd test
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

// TODO_IN_THIS_COMMIT: godoc & move...
func NewArmoredJson(kdf, salt, hint, ciphertext string) ArmoredJson {
	return ArmoredJson{
		Kdf:        kdf,
		Salt:       salt,
		SecParam:   strconv.Itoa(BcryptSecurityParameter),
		Hint:       hint,
		Ciphertext: ciphertext,
	}
}

// TODO_IN_THIS_COMMIT: godoc & move...
// Encrypt and armor the private key.
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

// ArmoredJson is a data structure which is used to (de)serialize the encrypted exported Morse private key file.
type ArmoredJson struct {
	Kdf        string `json:"kdf" yaml:"kdf"`
	Salt       string `json:"salt" yaml:"salt"`
	SecParam   string `json:"secparam" yaml:"secparam"`
	Hint       string `json:"hint" yaml:"hint"`
	Ciphertext string `json:"ciphertext" yaml:"ciphertext"`
}
