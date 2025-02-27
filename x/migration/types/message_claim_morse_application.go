package types

import (
	errorsmod "cosmossdk.io/errors"
	cometcrypto "github.com/cometbft/cometbft/crypto/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/gogoproto/proto"

	"github.com/pokt-network/poktroll/app/volatile"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ sdk.Msg = (*MsgClaimMorseApplication)(nil)

// NewMsgClaimMorseApplication creates a new MsgClaimMorseApplication.
// If morsePrivateKey is provided (i.e. not nil), it is used to sign the message.
func NewMsgClaimMorseApplication(
	shannonDestAddress string,
	morseSrcAddress string,
	morsePrivateKey cometcrypto.PrivKey,
	stake *sdk.Coin,
	serviceConfig *sharedtypes.ApplicationServiceConfig,
) (*MsgClaimMorseApplication, error) {
	msg := &MsgClaimMorseApplication{
		ShannonDestAddress: shannonDestAddress,
		MorseSrcAddress:    morseSrcAddress,
		Stake:              stake,
		ServiceConfig:      serviceConfig,
	}

	if morsePrivateKey != nil {
		if err := msg.SignMorseSignature(morsePrivateKey); err != nil {
			return nil, err
		}
	}

	return msg, nil
}

// ValidateBasic ensures that:
// - The morseSignature is non-empty (additional validation performed elsewhere).
// - The morseSrcAddress is valid (i.e. it is a valid hex-encoded address).
// - The shannonDestAddress is valid (i.e. it is a valid bech32 address).
func (msg *MsgClaimMorseApplication) ValidateBasic() error {
	if len(msg.MorseSignature) == 0 {
		return ErrMorseApplicationClaim.Wrap("morseSignature is empty")
	}

	if len(msg.MorseSrcAddress) != MorseAddressHexLengthBytes {
		return ErrMorseApplicationClaim.Wrapf("invalid morseSrcAddress length (%d)", len(msg.MorseSrcAddress))
	}

	if _, err := sdk.AccAddressFromBech32(msg.ShannonDestAddress); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid shannonDestAddress address (%s)", err)
	}

	// If msg.Stake is nil, it will default to the sake amount recorded in the corresponding MorseClaimableAccount.
	if msg.Stake != nil {
		if msg.Stake.Denom != volatile.DenomuPOKT {
			return ErrMorseApplicationClaim.Wrapf("invalid stake denom (%s)", msg.Stake.Denom)
		}

		if msg.Stake.IsValid() && msg.Stake.IsZero() {
			return ErrMorseApplicationClaim.Wrapf("invalid stake amount (%s)", msg.Stake.String())
		}
	}

	if err := sharedtypes.ValidateAppServiceConfigs([]*sharedtypes.ApplicationServiceConfig{
		msg.ServiceConfig,
	}); err != nil {
		return ErrMorseApplicationClaim.Wrapf("invalid service config: %s", err)
	}

	return nil
}

// SignMorseSignature signs the given MsgClaimMorseApplication with the given Morse private key.
func (msg *MsgClaimMorseApplication) SignMorseSignature(morsePrivKey cometcrypto.PrivKey) (err error) {
	signingMsgBz, err := msg.getSigningBytes()
	if err != nil {
		return err
	}

	msg.MorseSignature, err = morsePrivKey.Sign(signingMsgBz)
	return err
}

// ValidateMorseSignature validates the signature of the given MsgClaimMorseApplication
// matches the given Morse public key.
func (msg *MsgClaimMorseApplication) ValidateMorseSignature(morsePublicKey cometcrypto.PubKey) error {
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

// getSigningBytes returns the canonical byte representation of the MsgClaimMorseApplication
// which is used for signing and/or signature validation.
func (msg *MsgClaimMorseApplication) getSigningBytes() ([]byte, error) {
	// Copy msg and clear the morse signature field (ONLY on the copy) to prevent
	// it from being included in the signature validation.
	signingMsg := *msg
	signingMsg.MorseSignature = nil

	return proto.Marshal(&signingMsg)
}
