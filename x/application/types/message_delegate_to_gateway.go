package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgDelegateToGateway{}

func NewMsgDelegateToGateway(appAddress string, gatewayAddress string) *MsgDelegateToGateway {
	return &MsgDelegateToGateway{
		AppAddress:     appAddress,
		GatewayAddress: gatewayAddress,
	}
}

func (msg *MsgDelegateToGateway) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.AppAddress)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid appAddress address (%s)", err)
	}
	return nil
}
