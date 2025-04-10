package types

import (
	"crypto/ed25519"

	cmted25519 "github.com/cometbft/cometbft/crypto/ed25519"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
)

// morseClaimMessage is an interface that all Morse account/actor claim messages
// implement which allows for a non-generic implementations of common behavior.
type morseClaimMessage interface {
	cosmostypes.Msg

	getSigningBytes() ([]byte, error)

	GetMorsePublicKey() ed25519.PublicKey
	GetMorseSrcAddress() string
	GetMorseSignature() []byte
	ValidateMorseAddress() error
	ValidateMorseSignature() error
}

// validateMorseAddress validates that the morseSrcAddress matches
// the Morse public key of the given Morse claim message.
func validateMorseAddress(msg morseClaimMessage) error {
	publicKeyBz := msg.GetMorsePublicKey()
	if publicKeyBz == nil {
		return ErrMorseAccountClaim.Wrapf("morsePublicKey is nil")
	}

	publicKey := cmted25519.PubKey(publicKeyBz)

	if msg.GetMorseSrcAddress() != publicKey.Address().String() {
		return ErrMorseSrcAddress.Wrapf(
			"morseSrcAddress (%s) does not match morsePublicKey address (%s)",
			msg.GetMorseSrcAddress(),
			publicKey.Address().String(),
		)
	}
	return nil
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

	morsePublicKey := cmted25519.PubKey(msg.GetMorsePublicKey())

	signingMsgBz, err := msg.getSigningBytes()
	if err != nil {
		return err
	}

	// Validate the morse signature.
	if !morsePublicKey.VerifySignature(signingMsgBz, msg.GetMorseSignature()) {
		return ErrMorseSignature.Wrapf(
			"morseSignature (%x) is invalid for Morse address (%s)",
			msg.GetMorseSignature(),
			msg.GetMorseSrcAddress(),
		)
	}

	return nil
}
