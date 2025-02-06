package types

import (
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
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid shannonDestAddress address (%s)", err)
	}
	return nil
}
