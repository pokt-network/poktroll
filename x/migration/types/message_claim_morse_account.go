package types

import (
	cometcrypto "github.com/cometbft/cometbft/crypto"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/gogoproto/proto"
)

var (
	_ sdk.Msg           = (*MsgClaimMorseAccount)(nil)
	_ morseClaimMessage = (*MsgClaimMorseAccount)(nil)
)

func NewMsgClaimMorseAccount(
	shannonDestAddress string,
	morsePrivateKey cometcrypto.PrivKey,
	shannonSigningAddr string,
) (*MsgClaimMorseAccount, error) {
	msg := &MsgClaimMorseAccount{
		ShannonDestAddress:    shannonDestAddress,
		ShannonSigningAddress: shannonSigningAddr,
	}

	if morsePrivateKey != nil {
		msg.MorsePublicKey = morsePrivateKey.PubKey().Bytes()

		if err := msg.SignMsgClaimMorseAccount(morsePrivateKey); err != nil {
			return nil, err
		}
	}

	return msg, nil
}

// ValidateBasic ensures that:
// - The shannonDestAddress is valid (i.e. it is a valid bech32 address).
// - The morsePublicKey is valid.
// - The morseSrcAddress matches the public key.
// - The morseSignature is valid.
func (msg *MsgClaimMorseAccount) ValidateBasic() error {
	// Validate the shannonDestAddress is a valid bech32 address.
	if _, err := sdk.AccAddressFromBech32(msg.GetShannonDestAddress()); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf(
			"invalid shannonDestAddress address (%s): %s",
			msg.GetShannonDestAddress(), err,
		)
	}

	// Validate the Morse signature.
	if err := msg.ValidateMorseSignature(); err != nil {
		return err
	}
	return nil
}

// SignMsgClaimMorseAccount signs the given MsgClaimMorseAccount with the given Morse private key.
func (msg *MsgClaimMorseAccount) SignMsgClaimMorseAccount(morsePrivKey cometcrypto.PrivKey) (err error) {
	signingMsgBz, err := msg.getSigningBytes()
	if err != nil {
		return ErrMorseSignature.Wrapf("unable to get signing bytes: %s", err)
	}

	msg.MorseSignature, err = morsePrivKey.Sign(signingMsgBz)
	if err != nil {
		return ErrMorseSignature.Wrapf("unable to sign message: %s", err)
	}
	return nil
}

// ValidateMorseSignature validates the signature of the given MsgClaimMorseAccount
// matches the given Morse public key.
func (msg *MsgClaimMorseAccount) ValidateMorseSignature() error {
	return validateMorseSignature(msg)
}

// getSigningBytes returns the canonical byte representation of the MsgClaimMorseAccount
// which is used for signing and/or signature validation.
func (msg *MsgClaimMorseAccount) getSigningBytes() ([]byte, error) {
	// Copy msg and clear the morse signature field (ONLY on the copy) to prevent
	// it from being included in the signature validation.
	signingMsg := *msg
	signingMsg.MorseSignature = nil

	return proto.Marshal(&signingMsg)
}

// GetMorseSrcAddress returns the morse source address associated with
// the Morse public key of the given message.
func (msg *MsgClaimMorseAccount) GetMorseSrcAddress() string {
	return msg.GetMorsePublicKey().Address().String()
}
