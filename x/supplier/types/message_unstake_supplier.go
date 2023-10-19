package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
		// TODO(@Olshansk): Replace with a proper error
		return sdkerrors.Wrapf(ErrSample, "invalid address address (%s)", err)
	}
	return nil
}
