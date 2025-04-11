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

// validateMorseSignature validates the msg.morseSignature of the given morseClaimMessage.
// It checks that:
// - the morseSignature is the correct length
// - the morseSignature is valid for the signing bytes of the message associated with the public key
func validateMorseSignature(msg morseClaimMessage) error {
	if len(msg.GetMorseSignature()) != MorseSignatureLengthBytes {
		return ErrMorseSignature.Wrapf(
			"invalid morse signature length; expected %d, got %d",
			MorseSignatureLengthBytes, len(msg.GetMorseSignature()),
		)
	}

	signingMsgBz, err := msg.getSigningBytes()
	if err != nil {
		return err
	}

	// Validate the morse signature.
	if !msg.GetMorsePublicKey().VerifySignature(signingMsgBz, msg.GetMorseSignature()) {
		return ErrMorseSignature.Wrapf(
			"morseSignature (%x) is invalid for Morse address (%s)",
			msg.GetMorseSignature(),
			msg.GetMorseSrcAddress(),
		)
	}

	return nil
}

const MorseSignatureLengthBytes = 64
