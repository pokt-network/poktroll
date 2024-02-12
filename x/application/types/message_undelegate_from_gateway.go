package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = (*MsgUndelegateFromGateway)(nil)

func NewMsgUndelegateFromGateway(appAddress string, gatewayAddress string) *MsgUndelegateFromGateway {
	return &MsgUndelegateFromGateway{
		AppAddress:     appAddress,
		GatewayAddress: gatewayAddress,
	}
}

func (msg *MsgUndelegateFromGateway) NewRedelegationEvent() *EventRedelegation {
	return &EventRedelegation{
		AppAddress:     msg.AppAddress,
		GatewayAddress: msg.GatewayAddress,
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
