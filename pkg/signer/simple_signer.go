package signer

import (
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
)

var _ Signer = (*SimpleSigner)(nil)

// SimpleSigner is a signer implementation that uses the local keyring to sign
// messages, for verification using the signer's corresponding public key. Uses key name to sign.
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

// SimpleSignerByAddress is a signer implementation that uses the local keyring to sign
// messages, for verification using the signer's corresponding public key. Uses address to sign.
type SimpleSignerByAddress struct {
	keyring keyring.Keyring
	address string
}

// NewSimpleSigner creates a new SimpleSigner instance with the keyring and keyName provided
func NewSimpleSignerByAddress(keyring keyring.Keyring, address string) *SimpleSignerByAddress {
	return &SimpleSignerByAddress{keyring: keyring, address: address}
}

// Sign signs the given message using the SimpleSigner's keyring and keyName
func (s *SimpleSignerByAddress) Sign(msg [32]byte) (signature []byte, err error) {
	addr, err := cosmostypes.AccAddressFromBech32(s.address)
	if err != nil {
		return signature, err
	}

	sig, _, err := s.keyring.SignByAddress(addr, msg[:], signingtypes.SignMode_SIGN_MODE_DIRECT)
	return sig, err
}
