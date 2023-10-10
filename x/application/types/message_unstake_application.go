package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgUnstakeApplication = "unstake_application"

var _ sdk.Msg = &MsgUnstakeApplication{}

func NewMsgUnstakeApplication(address string) *MsgUnstakeApplication {
	return &MsgUnstakeApplication{
		Address: address,
	}
}

func (msg *MsgUnstakeApplication) Route() string {
	return RouterKey
}

func (msg *MsgUnstakeApplication) Type() string {
	return TypeMsgUnstakeApplication
}

func (msg *MsgUnstakeApplication) GetSigners() []sdk.AccAddress {
	address, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{address}
}

func (msg *MsgUnstakeApplication) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgUnstakeApplication) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid address address (%s)", err)
	}
	return nil
}
