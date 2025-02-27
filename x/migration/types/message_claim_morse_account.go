package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = (*MsgClaimMorseAccount)(nil)

func NewMsgClaimMorseAccount(shannonDestAddress string, morseSrcAddress string, morseSignature string) *MsgClaimMorseAccount {
	return &MsgClaimMorseAccount{
		ShannonDestAddress: shannonDestAddress,
		MorseSrcAddress:    morseSrcAddress,
		MorseSignature:     morseSignature,
	}
}

func (msg *MsgClaimMorseAccount) ValidateBasic() error {

	// TODO_UPNEXT(@bryanchriswhite#1034): Add validation

	_, err := sdk.AccAddressFromBech32(msg.ShannonDestAddress)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid shannonDestAddress address (%s)", err)
	}
	return nil
}
