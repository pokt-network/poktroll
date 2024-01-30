package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	_ sdk.Msg              = (*MsgUnstakeGateway)(nil)
	_ sdk.HasValidateBasic = (*MsgUnstakeGateway)(nil)
)

// NewMsgUnstakeGateway creates a new MsgUnstakeGateway message.
func NewMsgUnstakeGateway(address string) *MsgUnstakeGateway {
	return &MsgUnstakeGateway{Address: address}
}

// ValidateBasic validates the fields of the unstake message.
func (msg *MsgUnstakeGateway) ValidateBasic() error {
	// Validate the address
	_, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return ErrGatewayInvalidAddress.Wrapf(
			"invalid gateway address %s; (%v)",
			msg.Address, err,
		)
	}
	return nil
}
