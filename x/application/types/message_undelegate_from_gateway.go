package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgUndelegateFromGateway = "undelegate_from_gateway"

var _ sdk.Msg = &MsgUndelegateFromGateway{}

func NewMsgUndelegateFromGateway(address string) *MsgUndelegateFromGateway {
	return &MsgUndelegateFromGateway{
		Address: address,
	}
}

func (msg *MsgUndelegateFromGateway) Route() string {
	return RouterKey
}

func (msg *MsgUndelegateFromGateway) Type() string {
	return TypeMsgUndelegateFromGateway
}

func (msg *MsgUndelegateFromGateway) GetSigners() []sdk.AccAddress {
	address, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{address}
}

func (msg *MsgUndelegateFromGateway) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgUndelegateFromGateway) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid address address (%s)", err)
	}
	return nil
}
