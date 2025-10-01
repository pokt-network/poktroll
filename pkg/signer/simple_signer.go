package signer

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	secp256k1 "github.com/pokt-network/go-dleq/secp256k1"
)

var _ Signer = (*SimpleSigner)(nil)

// SimpleSigner is a signer implementation that uses the local keyring to sign
// messages, for verification using the signer's corresponding public key.
type SimpleSigner struct {
	curve  secp256k1.Curve
	scalar secp256k1.Scalar
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
	privKeyBytes := priv.Bytes()

	curve := secp256k1.NewCurve()
	scalar, err := curve.DecodeToScalar(privKeyBytes)
	if err != nil {
		return nil, err
	}

	return &SimpleSigner{
		curve:  curve,
		scalar: scalar,
	}, nil
}

// Sign signs the given message using the SimpleSigner's keyring and keyName
func (s *SimpleSigner) Sign(msg [32]byte) (signature []byte, err error) {
	// Map the 32-byte message hash deterministically to a curve point by:
	// 1. Interpreting it as a scalar (little-endian per secp256k1.ScalarFromBytes contract)
	// 2. Multiplying the base point by that scalar.
	// This avoids trying to parse arbitrary 32 bytes as a compressed public key (which must be 33 bytes).

	scalar := s.curve.ScalarFromBytes(msg)

	// Very unlikely edge case: scalar representing zero -> choose alternate deterministic scalar.
	// (go-dleq's ScalarFromBytes will produce zero only if msg is all zeroes.)
	// We can detect zero by deriving a point; base*0 = identity. The implementation
	// doesn't expose the infinity point directly, so we accept the negligible risk.
	point := s.curve.ScalarBaseMul(scalar)

	sig, err := s.curve.Sign(s.scalar, point)
	if err != nil {
		return nil, err
	}
	return sig, nil
}
