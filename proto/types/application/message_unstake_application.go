package application

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = (*MsgUnstakeApplication)(nil)

func NewMsgUnstakeApplication(appAddr string) *MsgUnstakeApplication {
	return &MsgUnstakeApplication{
		Address: appAddr,
	}
}

func (msg *MsgUnstakeApplication) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Address); err != nil {
		return ErrAppInvalidAddress.Wrapf("invalid address address (%s)", err)
	}
	return nil
}
