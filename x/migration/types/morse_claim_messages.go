package types

import (
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
)

// morseClaimMessage is an interface that all Morse account/actor claim messages
// implement which allows for a non-generic implementations of common behavior.
//
// Morse account/actor claim messages:
// - MsgClaimMorseAccount
// - MsgClaimMorseMultiSigAccount
// - MsgClaimMorseApplication
// - MsgClaimMorseSupplier
type MorseClaimMessage interface {
	cosmostypes.Msg

	getSigningBytes() ([]byte, error)

	GetShannonDestAddress() string
	GetMorsePublicKeyBz() []byte // if multisig, this is the amino-encoded list of public keys else ec25519 pubkey
	GetMorseSrcAddress() string
	GetMorseSignature() []byte // if multisig, this is the concatenated signature of all keys
	ValidateMorseSignature() error
	ValidateBasic() error
}

const MorseSignatureLengthBytes = 64
