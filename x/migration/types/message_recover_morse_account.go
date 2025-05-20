package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = (*MsgRecoverMorseAccount)(nil)

func NewMsgRecoverMorseAccount(authority string, shannonDestAddress string, morseSrcAddress string) *MsgRecoverMorseAccount {
	return &MsgRecoverMorseAccount{
		Authority:          authority,
		ShannonDestAddress: shannonDestAddress,
		MorseSrcAddress:    morseSrcAddress,
	}
}

func (msg *MsgRecoverMorseAccount) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid authority address (%s)", err)
	}

	if _, err := sdk.AccAddressFromBech32(msg.ShannonDestAddress); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid shannon destination address (%s)", err)
	}

	return nil
}
