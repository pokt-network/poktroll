package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = (*MsgUnstakeGateway)(nil)

func NewMsgUnstakeGateway(address string) *MsgUnstakeGateway {
	return &MsgUnstakeGateway{
		Address: address,
	}
}

func (msg *MsgUnstakeGateway) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return sdkerrors.Wrapf(ErrGatewayInvalidAddress, "invalid gateway address %s; (%v)", msg.Address, err)
	}
	return nil
}
