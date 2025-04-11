package types

import (
	errorsmod "cosmossdk.io/errors"
	cometcrypto "github.com/cometbft/cometbft/crypto"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/gogoproto/proto"
)

var _ sdk.Msg = (*MsgClaimMorseAccount)(nil)

func NewMsgClaimMorseAccount(
	shannonDestAddress string,
	morseSrcAddress string,
	morsePrivateKey cometcrypto.PrivKey,
	shannonSigningAddr string,
) (*MsgClaimMorseAccount, error) {
	msg := &MsgClaimMorseAccount{
		ShannonDestAddress:    shannonDestAddress,
		MorseSrcAddress:       morseSrcAddress,
		ShannonSigningAddress: shannonSigningAddr,
	}

	if morsePrivateKey != nil {
		if err := msg.SignMsgClaimMorseAccount(morsePrivateKey); err != nil {
			return nil, err
		}
	}

	return msg, nil
}

func (msg *MsgClaimMorseAccount) ValidateBasic() error {
	if len(msg.MorseSrcAddress) != MorseAddressHexLengthBytes {
		return ErrMorseAccountClaim.Wrapf("invalid morseSrcAddress length (%d): %q", len(msg.MorseSrcAddress), msg.MorseSrcAddress)
	}

	if len(msg.MorseSignature) != MorseSignatureLengthBytes {
		return ErrMorseAccountClaim.Wrapf("invalid morseSignature length (%d): %q", len(msg.MorseSignature), msg.MorseSignature)
	}

	if _, err := sdk.AccAddressFromBech32(msg.ShannonDestAddress); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid shannonDestAddress address (%s): %s", msg.ShannonDestAddress, err)
	}
	return nil
}

// SignMsgClaimMorseAccount signs the given MsgClaimMorseAccount with the given Morse private key.
func (msg *MsgClaimMorseAccount) SignMsgClaimMorseAccount(morsePrivKey cometcrypto.PrivKey) (err error) {
	signingMsgBz, err := msg.getSigningBytes()
	if err != nil {
		return err
	}

	msg.MorseSignature, err = morsePrivKey.Sign(signingMsgBz)
	return err
}

// ValidateMorseSignature validates the signature of the given MsgClaimMorseAccount
// matches the given Morse public key.
func (msg *MsgClaimMorseAccount) ValidateMorseSignature(morsePublicKey cometcrypto.PubKey) error {
	signingMsgBz, err := msg.getSigningBytes()
	if err != nil {
		return err
	}

	// Validate the morse signature.
	if !morsePublicKey.VerifySignature(signingMsgBz, msg.MorseSignature) {
		return ErrMorseAccountClaim.Wrapf("morseSignature is invalid")
	}

	return nil
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
