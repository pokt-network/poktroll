package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgRecoverMorseAccount{}

func NewMsgRecoverMorseAccount(authority string, shannonDestAddress string, morseSrcAddress string) *MsgRecoverMorseAccount {
	return &MsgRecoverMorseAccount{
		Authority:          authority,
		ShannonDestAddress: shannonDestAddress,
		MorseSrcAddress:    morseSrcAddress,
	}
}

func (msg *MsgRecoverMorseAccount) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid authority address (%s)", err)
	}
	return nil
}
