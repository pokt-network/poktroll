package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgTransferApplicationStake{}

func NewMsgTransferApplicationStake(creator string, address string, beneficiary string) *MsgTransferApplicationStake {
  return &MsgTransferApplicationStake{
		Creator: creator,
    Address: address,
    Beneficiary: beneficiary,
	}
}

func (msg *MsgTransferApplicationStake) ValidateBasic() error {
  _, err := sdk.AccAddressFromBech32(msg.Creator)
  	if err != nil {
  		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
  	}
  return nil
}

