package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const TypeMsgDelegateToGateway = "delegate_to_gateway"

var _ sdk.Msg = (*MsgDelegateToGateway)(nil)

func NewMsgDelegateToGateway(appAddress, gatewayAddress string) *MsgDelegateToGateway {
	return &MsgDelegateToGateway{
		AppAddress:     appAddress,
		GatewayAddress: gatewayAddress,
	}
}

func (msg *MsgDelegateToGateway) Route() string {
	return RouterKey
}

func (msg *MsgDelegateToGateway) Type() string {
	return TypeMsgDelegateToGateway
}

func (msg *MsgDelegateToGateway) GetSigners() []sdk.AccAddress {
	address, err := sdk.AccAddressFromBech32(msg.AppAddress)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{address}
}

func (msg *MsgDelegateToGateway) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgDelegateToGateway) NewDelegateeChangeEvent() *EventDelegateeChange {
	return &EventDelegateeChange{
		AppAddress: msg.AppAddress,
	}
}

func (msg *MsgDelegateToGateway) ValidateBasic() error {
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
