package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgCreateMorseAccountClaim{}

func NewMsgCreateMorseAccountClaim(
	shannonDestAddress string,
	morseSrcAddress string,
	morseSignature string,

) *MsgCreateMorseAccountClaim {
	return &MsgCreateMorseAccountClaim{
		ShannonDestAddress: shannonDestAddress,
		MorseSrcAddress:    morseSrcAddress,
		MorseSignature:     morseSignature,
	}
}

func (msg *MsgCreateMorseAccountClaim) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.ShannonDestAddress)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid shannonDestAddress address (%s)", err)
	}
	return nil
}

var _ sdk.Msg = &MsgUpdateMorseAccountClaim{}

func NewMsgUpdateMorseAccountClaim(
	shannonDestAddress string,
	morseSrcAddress string,
	morseSignature string,

) *MsgUpdateMorseAccountClaim {
	return &MsgUpdateMorseAccountClaim{
		ShannonDestAddress: shannonDestAddress,
		MorseSrcAddress:    morseSrcAddress,
		MorseSignature:     morseSignature,
	}
}

func (msg *MsgUpdateMorseAccountClaim) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.ShannonDestAddress)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid shannonDestAddress address (%s)", err)
	}
	return nil
}

var _ sdk.Msg = &MsgDeleteMorseAccountClaim{}

func NewMsgDeleteMorseAccountClaim(
	shannonDestAddress string,
	morseSrcAddress string,

) *MsgDeleteMorseAccountClaim {
	return &MsgDeleteMorseAccountClaim{
		ShannonDestAddress: shannonDestAddress,
		MorseSrcAddress:    morseSrcAddress,
	}
}

func (msg *MsgDeleteMorseAccountClaim) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.ShannonDestAddress)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid shannonDestAddress address (%s)", err)
	}
	return nil
}
