package types

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const TypeMsgUnstakeGateway = "unstake_gateway"

var _ sdk.Msg = &MsgUnstakeGateway{}

func NewMsgUnstakeGateway(address string) *MsgUnstakeGateway {
	return &MsgUnstakeGateway{
		Address: address,
	}
}

func (msg *MsgUnstakeGateway) Route() string {
	return RouterKey
}

func (msg *MsgUnstakeGateway) Type() string {
	return TypeMsgUnstakeGateway
}

func (msg *MsgUnstakeGateway) GetSigners() []sdk.AccAddress {
	address, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{address}
}

func (msg *MsgUnstakeGateway) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgUnstakeGateway) ValidateBasic() error {
	// Validate the address
	_, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return errors.Wrapf(ErrGatewayInvalidAddress, "invalid gateway address %s; (%v)", msg.Address, err)
	}
	return nil
}
