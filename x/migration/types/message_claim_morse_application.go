package types

import (
	cometcrypto "github.com/cometbft/cometbft/crypto/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/gogoproto/proto"

	"github.com/pokt-network/poktroll/pkg/encoding"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var (
	_ sdk.Msg           = (*MsgClaimMorseApplication)(nil)
	_ morseClaimMessage = (*MsgClaimMorseApplication)(nil)
)

// NewMsgClaimMorseApplication creates a new MsgClaimMorseApplication.
// If morsePrivateKey is provided (i.e. not nil), it is used to sign the message.
func NewMsgClaimMorseApplication(
	shannonDestAddress string,
	morsePrivateKey cometcrypto.PrivKey,
	serviceConfig *sharedtypes.ApplicationServiceConfig,
	shannonSigningAddr string,
) (*MsgClaimMorseApplication, error) {
	msg := &MsgClaimMorseApplication{
		ShannonDestAddress:    shannonDestAddress,
		ServiceConfig:         serviceConfig,
		ShannonSigningAddress: shannonSigningAddr,
	}

	if morsePrivateKey != nil {
		msg.MorsePublicKey = morsePrivateKey.PubKey().Bytes()

		if err := msg.SignMorseSignature(morsePrivateKey); err != nil {
			return nil, err
		}
	}

	return msg, nil
}

// ValidateBasic ensures that:
// - The shannonDestAddress is valid (i.e. it is a valid bech32 address).
// - The application service config is valid.
// - The morsePublicKey is valid.
// - The morseSrcAddress matches the public key.
// - The morseSignature is valid.
func (msg *MsgClaimMorseApplication) ValidateBasic() error {
	// Validate the shannonDestAddress is a valid bech32 address.
	if _, err := sdk.AccAddressFromBech32(msg.GetShannonDestAddress()); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf(
			"invalid shannonDestAddress address (%s): %s",
			msg.GetShannonDestAddress(), err,
		)
	}

	// Validate the application service config.
	if err := sharedtypes.ValidateAppServiceConfigs([]*sharedtypes.ApplicationServiceConfig{
		msg.ServiceConfig,
	}); err != nil {
		return ErrMorseApplicationClaim.Wrapf("invalid service config: %s", err)
	}

	// Validate the Morse signature.
	if err := msg.ValidateMorseSignature(); err != nil {
		return err
	}
	return nil
}

// SignMorseSignature signs the given MsgClaimMorseApplication with the given Morse private key.
func (msg *MsgClaimMorseApplication) SignMorseSignature(morsePrivKey cometcrypto.PrivKey) (err error) {
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

// ValidateMorseSignature validates the signature of the given MsgClaimMorseApplication
// matches the given Morse public key.
func (msg *MsgClaimMorseApplication) ValidateMorseSignature() error {
	return validateMorseSignature(msg)
}

// getSigningBytes returns the canonical byte representation of the MsgClaimMorseApplication
// which is used for signing and/or signature validation.
func (msg *MsgClaimMorseApplication) getSigningBytes() ([]byte, error) {
	// Copy msg and clear the morse signature field (ONLY on the copy) to prevent
	// it from being included in the signature validation.
	signingMsg := *msg
	signingMsg.MorseSignature = nil

	return proto.Marshal(&signingMsg)
}

// GetMorseSignerAddress returns the morse address which was used to sign the claim message.
func (msg *MsgClaimMorseApplication) GetMorseSignerAddress() string {
	return encoding.NormalizeMorseAddress(msg.GetMorsePublicKey().Address().String())
}
