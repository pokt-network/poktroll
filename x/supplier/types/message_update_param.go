package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgUpdateParam{}

func NewMsgUpdateParam(authority string, name string, asType string) *MsgUpdateParam {
  return &MsgUpdateParam{
		Authority: authority,
    Name: name,
    AsType: asType,
	}
}

func (msg *MsgUpdateParam) ValidateBasic() error {
  _, err := sdk.AccAddressFromBech32(msg.Authority)
  	if err != nil {
  		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid authority address (%s)", err)
  	}
  return nil
}

