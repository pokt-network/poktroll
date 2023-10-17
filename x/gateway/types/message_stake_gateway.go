package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgStakeGateway = "stake_gateway"

var _ sdk.Msg = &MsgStakeGateway{}

func NewMsgStakeGateway(address string) *MsgStakeGateway {
	return &MsgStakeGateway{
		Address: address,
	}
}

func (msg *MsgStakeGateway) Route() string {
	return RouterKey
}

func (msg *MsgStakeGateway) Type() string {
	return TypeMsgStakeGateway
}

func (msg *MsgStakeGateway) GetSigners() []sdk.AccAddress {
	address, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{address}
}

func (msg *MsgStakeGateway) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgStakeGateway) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid address address (%s)", err)
	}
	return nil
}
