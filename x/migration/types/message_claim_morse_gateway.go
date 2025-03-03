package types

import (
	errorsmod "cosmossdk.io/errors"
	cometcrypto "github.com/cometbft/cometbft/crypto"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/gogoproto/proto"

	"github.com/pokt-network/poktroll/app/volatile"
)

var _ sdk.Msg = &MsgClaimMorseGateway{}

// NewMsgClaimMorseGateway creates a new MsgClaimMorseGateway.
// If morsePrivateKey is provided (i.e. not nil), it is used to sign the message.
func NewMsgClaimMorseGateway(
	shannonDestAddress string,
	morseSrcAddress string,
	morsePrivateKey cometcrypto.PrivKey,
	stake sdk.Coin,
) (*MsgClaimMorseGateway, error) {
	msg := &MsgClaimMorseGateway{
		ShannonDestAddress: shannonDestAddress,
		MorseSrcAddress:    morseSrcAddress,
		Stake:              stake,
	}

	if morsePrivateKey != nil {
		if err := msg.SignMorseSignature(morsePrivateKey); err != nil {
			return nil, err
		}
	}

	return msg, nil
}

// ValidateBasic ensures that:
// - The morseSignature length is valid (signature validation performed elsewhere).
// - The morseSrcAddress is valid (i.e. it is a valid hex-encoded address).
// - The shannonDestAddress is valid (i.e. it is a valid bech32 address).
func (msg *MsgClaimMorseGateway) ValidateBasic() error {
	if len(msg.MorseSignature) != MorseSignatureLengthBytes {
		return ErrMorseGatewayClaim.Wrapf(
			"invalid morse signature length; expected %d, got %d",
			MorseSignatureLengthBytes, len(msg.MorseSignature),
		)
	}

	if len(msg.MorseSrcAddress) != MorseAddressHexLengthBytes {
		return ErrMorseGatewayClaim.Wrapf("invalid morseSrcAddress length (%d)", len(msg.MorseSrcAddress))
	}

	if _, err := sdk.AccAddressFromBech32(msg.ShannonDestAddress); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid shannonDestAddress address (%s)", err)
	}

	// If msg.Stake is nil, it will default to the sake amount recorded in the corresponding MorseClaimableAccount.
	if msg.Stake.Denom != volatile.DenomuPOKT {
		return ErrMorseGatewayClaim.Wrapf("invalid stake denom (%s)", msg.Stake.Denom)
	}

	if msg.Stake.IsValid() && msg.Stake.IsZero() {
		return ErrMorseGatewayClaim.Wrapf("invalid stake amount (%s)", msg.Stake.String())
	}

	return nil
}

// SignMorseSignature signs the given MsgClaimMorseGateway with the given Morse private key.
func (msg *MsgClaimMorseGateway) SignMorseSignature(morsePrivKey cometcrypto.PrivKey) (err error) {
	signingMsgBz, err := msg.getSigningBytes()
	if err != nil {
		return err
	}

	msg.MorseSignature, err = morsePrivKey.Sign(signingMsgBz)
	return err
}

// ValidateMorseSignature validates the signature of the given MsgClaimMorseGateway
// matches the given Morse public key.
func (msg *MsgClaimMorseGateway) ValidateMorseSignature(morsePublicKey cometcrypto.PubKey) error {
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

// getSigningBytes returns the canonical byte representation of the MsgClaimMorseGateway
// which is used for signing and/or signature validation.
func (msg *MsgClaimMorseGateway) getSigningBytes() ([]byte, error) {
	// Copy msg and clear the morse signature field (ONLY on the copy) to prevent
	// it from being included in the signature validation.
	signingMsg := *msg
	signingMsg.MorseSignature = nil

	return proto.Marshal(&signingMsg)
}
