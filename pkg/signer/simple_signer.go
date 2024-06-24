package signer

import (
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
)

var _ Signer = (*SimpleSigner)(nil)

// SimpleSigner is a signer implementation that uses the local keyring to sign
// messages, for verification using the signer's corresponding public key.
type SimpleSigner struct {
	keyring keyring.Keyring
	keyName string
}

// NewSimpleSigner creates a new SimpleSigner instance with the keyring and keyName provided
func NewSimpleSigner(keyring keyring.Keyring, keyName string) *SimpleSigner {
	return &SimpleSigner{keyring: keyring, keyName: keyName}
}

// Sign signs the given message using the SimpleSigner's keyring and keyName
func (s *SimpleSigner) Sign(msg [32]byte) (signature []byte, err error) {
	sig, _, err := s.keyring.Sign(s.keyName, msg[:], signingtypes.SignMode_SIGN_MODE_DIRECT)
	return sig, err
}
