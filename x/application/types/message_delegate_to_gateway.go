package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgDelegateToGateway = "delegate_to_gateway"

var _ sdk.Msg = &MsgDelegateToGateway{}

func NewMsgDelegateToGateway(address string) *MsgDelegateToGateway {
	return &MsgDelegateToGateway{
		Address: address,
	}
}

func (msg *MsgDelegateToGateway) Route() string {
	return RouterKey
}

func (msg *MsgDelegateToGateway) Type() string {
	return TypeMsgDelegateToGateway
}

func (msg *MsgDelegateToGateway) GetSigners() []sdk.AccAddress {
	address, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{address}
}

func (msg *MsgDelegateToGateway) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgDelegateToGateway) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid address address (%s)", err)
	}
	return nil
}
