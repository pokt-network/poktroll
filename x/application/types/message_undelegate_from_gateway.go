package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgUndelegateFromGateway{}

func NewMsgUndelegateFromGateway(appAddress string, gatewayAddress string) *MsgUndelegateFromGateway {
	return &MsgUndelegateFromGateway{
		AppAddress:     appAddress,
		GatewayAddress: gatewayAddress,
	}
}

func (msg *MsgUndelegateFromGateway) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.AppAddress)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid appAddress address (%s)", err)
	}
	return nil
}
