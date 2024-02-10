package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgUnstakeApplication{}

func NewMsgUnstakeApplication(address string) *MsgUnstakeApplication {
	return &MsgUnstakeApplication{
		Address: address,
	}
}

func (msg *MsgUnstakeApplication) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid address address (%s)", err)
	}
	return nil
}
