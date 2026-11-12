package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ sdk.Msg = (*MsgUnstakeApplication)(nil)

func NewMsgUnstakeApplication(appAddr string) *MsgUnstakeApplication {
	return &MsgUnstakeApplication{
		Address: appAddr,
	}
}

func (msg *MsgUnstakeApplication) ValidateBasic() error {
	if err := sharedtypes.ValidateCanonicalAddress(msg.Address); err != nil {
		return ErrAppInvalidAddress.Wrapf("invalid address address (%s)", err)
	}
	return nil
}
