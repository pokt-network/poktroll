package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgImportMorseClaimableAccounts{}

func NewMsgImportMorseClaimableAccounts(authority string, morseAccountState string) *MsgImportMorseClaimableAccounts {
  return &MsgImportMorseClaimableAccounts{
		Authority: authority,
    MorseAccountState: morseAccountState,
	}
}

func (msg *MsgImportMorseClaimableAccounts) ValidateBasic() error {
  _, err := sdk.AccAddressFromBech32(msg.Authority)
  	if err != nil {
  		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid authority address (%s)", err)
  	}
  return nil
}

