package signer

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
)

var _ Signer = (*SimpleSigner)(nil)

// SimpleSigner is a signer implementation that uses the local keyring to sign
// messages, for verification using the signer's corresponding public key.
type SimpleSigner struct {
	privKey cryptotypes.PrivKey
}

// NewSimpleSigner creates a new SimpleSigner instance with the keyring and keyName provided
func NewSimpleSigner(kr keyring.Keyring, keyName string) (*SimpleSigner, error) {
	// Resolve key info
	info, err := kr.Key(keyName)
	if err != nil {
		return nil, err
	}

	local := info.GetLocal()
	if local.PrivKey == nil {
		return nil, fmt.Errorf("private key is not available")
	}

	priv, ok := local.PrivKey.GetCachedValue().(cryptotypes.PrivKey)
	if !ok {
		return nil, fmt.Errorf("unable to cast to cryptotypes")
	}

	return &SimpleSigner{
		privKey: priv,
	}, nil
}

// Sign signs the given message using the SimpleSigner's keyring and keyName
func (s *SimpleSigner) Sign(msg [32]byte) (signature []byte, err error) {
	sig, err := s.privKey.Sign(msg[:])
	return sig, nil
}
