package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgUnstakeSupplier = "unstake_supplier"

var _ sdk.Msg = &MsgUnstakeSupplier{}

func NewMsgUnstakeSupplier(address string) *MsgUnstakeSupplier {
	return &MsgUnstakeSupplier{
		Address: address,
	}
}

func (msg *MsgUnstakeSupplier) Route() string {
	return RouterKey
}

func (msg *MsgUnstakeSupplier) Type() string {
	return TypeMsgUnstakeSupplier
}

func (msg *MsgUnstakeSupplier) GetSigners() []sdk.AccAddress {
	address, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{address}
}

func (msg *MsgUnstakeSupplier) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgUnstakeSupplier) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid address address (%s)", err)
	}
	return nil
}
