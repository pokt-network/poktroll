package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgUndelegateFromGateway = "undelegate_from_gateway"

var _ sdk.Msg = (*MsgUndelegateFromGateway)(nil)

func NewMsgUndelegateFromGateway(appAddress, gatewayAddress string) *MsgUndelegateFromGateway {
	return &MsgUndelegateFromGateway{
		AppAddress:     appAddress,
		GatewayAddress: gatewayAddress,
	}
}

func (msg *MsgUndelegateFromGateway) Route() string {
	return RouterKey
}

func (msg *MsgUndelegateFromGateway) Type() string {
	return TypeMsgUndelegateFromGateway
}

func (msg *MsgUndelegateFromGateway) GetSigners() []sdk.AccAddress {
	address, err := sdk.AccAddressFromBech32(msg.AppAddress)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{address}
}

func (msg *MsgUndelegateFromGateway) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgUndelegateFromGateway) NewDelegateeChangeEvent() *EventDelegateeChange {
	return &EventDelegateeChange{
		AppAddress: msg.AppAddress,
	}
}

func (msg *MsgUndelegateFromGateway) ValidateBasic() error {
	// Validate the application address
	if _, err := sdk.AccAddressFromBech32(msg.AppAddress); err != nil {
		return sdkerrors.Wrapf(ErrAppInvalidAddress, "invalid application address %s; (%v)", msg.AppAddress, err)
	}
	// Validate the gateway address
	if _, err := sdk.AccAddressFromBech32(msg.GatewayAddress); err != nil {
		return sdkerrors.Wrapf(ErrAppInvalidGatewayAddress, "invalid gateway address %s; (%v)", msg.GatewayAddress, err)
	}
	return nil
}
