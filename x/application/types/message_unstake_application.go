package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = (*MsgUnstakeApplication)(nil)

func NewMsgUnstakeApplication(address string) *MsgUnstakeApplication {
	return &MsgUnstakeApplication{
		Address: address,
	}
}

func (msg *MsgUnstakeApplication) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return errorsmod.Wrapf(ErrAppInvalidAddress, "invalid address address (%s)", err)
	}
	return nil
}
