package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgUnstakeGateway{}

func NewMsgUnstakeGateway(address string) *MsgUnstakeGateway {
	return &MsgUnstakeGateway{
		Address: address,
	}
}

func (msg *MsgUnstakeGateway) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return ErrGatewayInvalidAddress.Wrapf("invalid gateway address (%s); (%v)", msg.Address, err)
	}
	return nil
}
