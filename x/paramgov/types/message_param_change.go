package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgParamChange{}

func NewMsgParamChange(creator string, moduleName string, paramName string, value string, targetHeight int32) *MsgParamChange {
  return &MsgParamChange{
		Creator: creator,
    ModuleName: moduleName,
    ParamName: paramName,
    Value: value,
    TargetHeight: targetHeight,
	}
}

func (msg *MsgParamChange) ValidateBasic() error {
  _, err := sdk.AccAddressFromBech32(msg.Creator)
  	if err != nil {
  		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
  	}
  return nil
}

