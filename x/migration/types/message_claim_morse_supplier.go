package types

import (
	errorsmod "cosmossdk.io/errors"
	cometcrypto "github.com/cometbft/cometbft/crypto/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/gogoproto/proto"

	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ sdk.Msg = &MsgClaimMorseSupplier{}

func NewMsgClaimMorseSupplier(
	shannonDestAddress string,
	morseSrcAddress string,
	morsePrivateKey cometcrypto.PrivKey,
	serviceConfig *sharedtypes.SupplierServiceConfig,
) (*MsgClaimMorseSupplier, error) {
	msg := &MsgClaimMorseSupplier{
		ShannonDestAddress: shannonDestAddress,
		MorseSrcAddress:    morseSrcAddress,
		Services:           serviceConfig,
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
func (msg *MsgClaimMorseSupplier) ValidateBasic() error {
	if len(msg.MorseSignature) != MorseSignatureLengthBytes {
		return ErrMorseSupplierClaim.Wrapf(
			"invalid morse signature length; expected %d, got %d",
			MorseSignatureLengthBytes, len(msg.MorseSignature),
		)
	}

	if len(msg.MorseSrcAddress) != MorseAddressHexLengthBytes {
		return ErrMorseSupplierClaim.Wrapf("invalid morseSrcAddress length (%d)", len(msg.MorseSrcAddress))
	}

	if _, err := sdk.AccAddressFromBech32(msg.ShannonDestAddress); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid shannonDestAddress address (%s)", err)
	}

	if err := sharedtypes.ValidateSupplierServiceConfigs([]*sharedtypes.SupplierServiceConfig{
		msg.Services,
	}); err != nil {
		return ErrMorseSupplierClaim.Wrapf("invalid service config: %s", err)
	}

	return nil
}

// SignMorseSignature signs the given MsgClaimMorseApplication with the given Morse private key.
func (msg *MsgClaimMorseSupplier) SignMorseSignature(morsePrivKey cometcrypto.PrivKey) (err error) {
	signingMsgBz, err := msg.getSigningBytes()
	if err != nil {
		return err
	}

	msg.MorseSignature, err = morsePrivKey.Sign(signingMsgBz)
	return err
}

// ValidateMorseSignature validates the signature of the given MsgClaimMorseSupplier
// matches the given Morse public key.
func (msg *MsgClaimMorseSupplier) ValidateMorseSignature(morsePublicKey cometcrypto.PubKey) error {
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

// getSigningBytes returns the canonical byte representation of the MsgClaimMorseSupplier
// which is used for signing and/or signature validation.
func (msg *MsgClaimMorseSupplier) getSigningBytes() ([]byte, error) {
	// Copy msg and clear the morse signature field (ONLY on the copy) to prevent
	// it from being included in the signature validation.
	signingMsg := *msg
	signingMsg.MorseSignature = nil

	return proto.Marshal(&signingMsg)
}
