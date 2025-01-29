package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgCreateMorseAccountState{}

func NewMsgCreateMorseAccountState(authority string, morseAccountState MorseAccountState) *MsgCreateMorseAccountState {
	return &MsgCreateMorseAccountState{
		Authority:         authority,
		MorseAccountState: morseAccountState,
	}
}

func (msg *MsgCreateMorseAccountState) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
