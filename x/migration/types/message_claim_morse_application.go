package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgClaimMorseApplication{}

func NewMsgClaimMorseApplication(shannonDestAddress string, morseSrcAddress string, morseSignature string, stake sdk.Coin, serviceConfig string) *MsgClaimMorseApplication {
	// TODO_MAINNET(@bryanchriswhite, #1034): Add message signing.

	return &MsgClaimMorseApplication{
		ShannonDestAddress: shannonDestAddress,
		MorseSrcAddress:    morseSrcAddress,
		MorseSignature:     morseSignature,
		Stake:              stake,
		ServiceConfig:      serviceConfig,
	}
}

func (msg *MsgClaimMorseApplication) ValidateBasic() error {
	// TODO_MAINNET(@bryanchriswhite, #1034): Add validation.

	_, err := sdk.AccAddressFromBech32(msg.ShannonDestAddress)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid shannonDestAddress address (%s)", err)
	}
	return nil
}
