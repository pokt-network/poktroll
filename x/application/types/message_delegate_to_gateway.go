package types

import sdk "github.com/cosmos/cosmos-sdk/types"

var _ sdk.Msg = (*MsgDelegateToGateway)(nil)

func NewMsgDelegateToGateway(appAddress, gatewayAddress string) *MsgDelegateToGateway {
	return &MsgDelegateToGateway{
		AppAddress:     appAddress,
		GatewayAddress: gatewayAddress,
	}
}

func (msg *MsgDelegateToGateway) ValidateBasic() error {
	// Validate the application address
	if _, err := sdk.AccAddressFromBech32(msg.AppAddress); err != nil {
		return ErrAppInvalidAddress.Wrapf("invalid application address %s; (%v)", msg.AppAddress, err)
	}
	// Validate the gateway address
	if _, err := sdk.AccAddressFromBech32(msg.GatewayAddress); err != nil {
		return ErrAppInvalidGatewayAddress.Wrapf("invalid gateway address %s; (%v)", msg.GatewayAddress, err)
	}
	return nil
}
