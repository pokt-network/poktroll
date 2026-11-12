package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ sdk.Msg = (*MsgUnstakeGateway)(nil)

func NewMsgUnstakeGateway(address string) *MsgUnstakeGateway {
	return &MsgUnstakeGateway{
		Address: address,
	}
}

func (msg *MsgUnstakeGateway) ValidateBasic() error {
	err := sharedtypes.ValidateCanonicalAddress(msg.Address)
	if err != nil {
		return ErrGatewayInvalidAddress.Wrapf("invalid gateway address %s; (%v)", msg.Address, err)
	}
	return nil
}
