package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const TypeMsgStakeSupplier = "stake_supplier"

var _ sdk.Msg = &MsgStakeSupplier{}

func NewMsgStakeSupplier(address string) *MsgStakeSupplier {
	return &MsgStakeSupplier{
		Address: address,
	}
}

func (msg *MsgStakeSupplier) Route() string {
	return RouterKey
}

func (msg *MsgStakeSupplier) Type() string {
	return TypeMsgStakeSupplier
}

func (msg *MsgStakeSupplier) GetSigners() []sdk.AccAddress {
	address, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{address}
}

func (msg *MsgStakeSupplier) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgStakeSupplier) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		// TODO(@bryanchriswhite): Replace with a proper error
		return sdkerrors.Wrapf(ErrSample, "invalid address address (%s)", err)
	}
	return nil
}
