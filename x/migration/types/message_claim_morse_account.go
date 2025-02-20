package types

import (
	"encoding/hex"

	errorsmod "cosmossdk.io/errors"
	cometcrypto "github.com/cometbft/cometbft/crypto"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/gogoproto/proto"
)

var _ sdk.Msg = &MsgClaimMorseAccount{}

func NewMsgClaimMorseAccount(
	shannonDestAddress string,
	morseSrcAddress string,
	morsePrivateKey cometcrypto.PrivKey,
) (*MsgClaimMorseAccount, error) {
	msg := &MsgClaimMorseAccount{
		ShannonDestAddress: shannonDestAddress,
		MorseSrcAddress:    morseSrcAddress,
	}
	msgBz, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}

	morseSignature, err := morsePrivateKey.Sign(msgBz)
	if err != nil {
		return nil, err
	}

	msg.MorseSignature = hex.EncodeToString(morseSignature)

	return msg, nil
}

func (msg *MsgClaimMorseAccount) ValidateBasic() error {
	if len(msg.MorseSrcAddress) != MorseAddressHexLengthBytes {
		return ErrMorseAccountClaim.Wrapf("invalid morseSrcAddress length (%d)", len(msg.MorseSrcAddress))
	}

	if _, err := sdk.AccAddressFromBech32(msg.ShannonDestAddress); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid shannonDestAddress address (%s)", err)
	}
	return nil
}

// SignMorseSignature signs the given MsgClaimMorseAccount with the given Morse private key.
func (msg *MsgClaimMorseAccount) SignMorseSignature(morsePrivKey cometcrypto.PrivKey) error {
	signingMsgBz, err := msg.getSigningBytes()
	if err != nil {
		return err
	}

	signatureBz, err := morsePrivKey.Sign(signingMsgBz)
	if err != nil {
		return err
	}

	msg.MorseSignature = hex.EncodeToString(signatureBz)
	return nil
}

// ValidateMorseSignature validates the signature of the given MsgClaimMorseAccount
// matches the given Morse public key.
func (msg *MsgClaimMorseAccount) ValidateMorseSignature(morsePublicKey cometcrypto.PubKey) error {
	// Validate the morse signature.
	morseSignature, err := hex.DecodeString(msg.MorseSignature)
	if err != nil {
		return err
	}

	signingMsgBz, err := msg.getSigningBytes()
	if err != nil {
		return err
	}

	// Validate the morse signature.
	if !morsePublicKey.VerifySignature(signingMsgBz, morseSignature) {
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
	signingMsg.MorseSignature = ""

	return proto.Marshal(&signingMsg)
}
