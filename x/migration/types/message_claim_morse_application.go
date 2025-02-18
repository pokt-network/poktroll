package types

import (
	"encoding/hex"

	errorsmod "cosmossdk.io/errors"
	cometcrypto "github.com/cometbft/cometbft/crypto/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/gogoproto/proto"

	"github.com/pokt-network/poktroll/app/volatile"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ sdk.Msg = &MsgClaimMorseApplication{}

func NewMsgClaimMorseApplication(
	shannonDestAddress string,
	morseSrcAddress string,
	morsePrivateKey cometcrypto.PrivKey,
	stake sdk.Coin,
	serviceConfig *sharedtypes.ApplicationServiceConfig,
) (*MsgClaimMorseApplication, error) {
	msg := &MsgClaimMorseApplication{
		ShannonDestAddress: shannonDestAddress,
		MorseSrcAddress:    morseSrcAddress,
		Stake:              stake,
		ServiceConfig:      serviceConfig,
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

	if msg.Stake.Denom != volatile.DenomuPOKT {
		return ErrMorseApplicationClaim.Wrapf("invalid stake denom (%s)", msg.Stake.Denom)
	}

	if msg.Stake.IsValid() && msg.Stake.IsZero() {
		return ErrMorseApplicationClaim.Wrapf("invalid stake amount (%s)", msg.Stake.String())
	}

	if err := sharedtypes.ValidateAppServiceConfigs([]*sharedtypes.ApplicationServiceConfig{
		msg.ServiceConfig,
	}); err != nil {
		return ErrMorseApplicationClaim.Wrapf("invalid service config: %s", err)
	}

	return nil
}
