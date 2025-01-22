package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgUploadMorseState{}

func NewMsgUploadMorseState(authority string, state MorseAccountState) *MsgUploadMorseState {
	return &MsgUploadMorseState{
		Authority: authority,
		State:     state,
	}
}

func (msg *MsgUploadMorseState) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid authority address (%s)", err)
	}
	return nil
}
