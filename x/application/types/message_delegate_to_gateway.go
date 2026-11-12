package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ sdk.Msg = (*MsgDelegateToGateway)(nil)

func NewMsgDelegateToGateway(appAddress, gatewayAddress string) *MsgDelegateToGateway {
	return &MsgDelegateToGateway{
		AppAddress:     appAddress,
		GatewayAddress: gatewayAddress,
	}
}

func (msg *MsgDelegateToGateway) ValidateBasic() error {
	// Validate the application address
	if err := sharedtypes.ValidateCanonicalAddress(msg.AppAddress); err != nil {
		return ErrAppInvalidAddress.Wrapf("invalid application address %s; (%v)", msg.AppAddress, err)
	}
	// Validate the gateway address
	if err := sharedtypes.ValidateCanonicalAddress(msg.GatewayAddress); err != nil {
		return ErrAppInvalidGatewayAddress.Wrapf("invalid gateway address %s; (%v)", msg.GatewayAddress, err)
	}
	return nil
}
