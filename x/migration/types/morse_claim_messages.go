package types

import (
	cmted25519 "github.com/cometbft/cometbft/crypto/ed25519"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
)

// morseClaimMessage is an interface that all Morse account/actor claim messages
// implement which allows for a non-generic implementations of common behavior.
//
// Morse account/actor claim messages:
// - MsgClaimMorseAccount
// - MsgClaimMorseApplication
// - MsgClaimMorseSupplier
type morseClaimMessage interface {
	cosmostypes.Msg

	getSigningBytes() ([]byte, error)

	GetMorsePublicKey() cmted25519.PubKey
	GetMorseSrcAddress() string
	GetMorseSignature() []byte
	ValidateMorseSignature() error
}

type morseMultisigClaimMessage interface {
	cosmostypes.Msg

	getSigningBytes() ([]byte, error)

	GetMorseMultisigPublicKeys() []cmted25519.PubKey
	GetMorseSrcAddress() string
	GetMorseSignature() []byte
	ValidateMorseSignature() error
}

// validateMorseSignature validates the msg.morseSignature of the given morseClaimMessage.
// It checks that:
// - the morseSignature is the correct length
// - the morseSignature is valid for the signing bytes of the message associated with the public key
func validateMorseSignature(msg cosmostypes.Msg) error {
	switch m := msg.(type) {
	case morseClaimMessage:
		// existing single-key validation
		if len(m.GetMorseSignature()) != MorseSignatureLengthBytes {
			return ErrMorseSignature.Wrapf(
				"invalid morse signature length; expected %d, got %d",
				MorseSignatureLengthBytes, len(m.GetMorseSignature()),
			)
		}
		signingBz, err := m.getSigningBytes()
		if err != nil {
			return err
		}
		if !m.GetMorsePublicKey().VerifySignature(signingBz, m.GetMorseSignature()) {
			return ErrMorseSignature.Wrapf(
				"morseSignature (%x) is invalid for Morse address (%s)",
				m.GetMorseSignature(),
				m.GetMorseSrcAddress(),
			)
		}
		return nil

	case morseMultisigClaimMessage:
		return validateMorseMultisigSignature(m)

	default:
		return ErrMorseSignature.Wrap("message does not implement Morse claim interface")
	}
}

const MorseSignatureLengthBytes = 64
